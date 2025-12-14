package postgres

import (
	"context"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Migration struct {
	Version int64
	Path    string
	Name    string
	SQL     string
}

func RunMigrations(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	if dir == "" {
		dir = "internal/migrations"
	}

	migs, err := loadMigrations(dir)
	if err != nil {
		return fmt.Errorf("load migrations: %w", err)
	}
	if len(migs) == 0 {
		return nil
	}

	// Prevent concurrent migration runs (especially in docker compose restarts).
	lockID := advisoryLockID("demo_service_migrations")
	if _, err := pool.Exec(ctx, `SELECT pg_advisory_lock($1)`, lockID); err != nil {
		return fmt.Errorf("advisory lock: %w", err)
	}
	defer func() { _, _ = pool.Exec(context.Background(), `SELECT pg_advisory_unlock($1)`, lockID) }()

	if err := ensureMigrationsTable(ctx, pool); err != nil {
		return err
	}

	applied, err := appliedVersions(ctx, pool)
	if err != nil {
		return err
	}

	for _, m := range migs {
		if applied[m.Version] {
			continue
		}

		// Each migration in its own tx.
		tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			return fmt.Errorf("begin tx (v=%d): %w", m.Version, err)
		}

		if _, err := tx.Exec(ctx, m.SQL); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("exec migration (v=%d, %s): %w", m.Version, m.Name, err)
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO schema_migrations (version, name, applied_at)
			VALUES ($1, $2, now())
		`, m.Version, m.Name); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("record migration (v=%d): %w", m.Version, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit migration (v=%d): %w", m.Version, err)
		}
	}

	return nil
}

func ensureMigrationsTable(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    BIGINT PRIMARY KEY,
			name       TEXT NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	return nil
}

func appliedVersions(ctx context.Context, pool *pgxpool.Pool) (map[int64]bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := pool.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("query schema_migrations: %w", err)
	}
	defer rows.Close()

	out := make(map[int64]bool)
	for rows.Next() {
		var v int64
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("scan version: %w", err)
		}
		out[v] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows err: %w", err)
	}
	return out, nil
}

func loadMigrations(dir string) ([]Migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("readdir %s: %w", dir, err)
	}

	var migs []Migration
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".up.sql") {
			continue
		}

		// Expect: 0001_init.up.sql  -> version=1
		version, ok := parseVersion(name)
		if !ok {
			return nil, fmt.Errorf("invalid migration filename: %s (expected like 0001_name.up.sql)", name)
		}

		path := filepath.Join(dir, name)
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}

		migs = append(migs, Migration{
			Version: version,
			Path:    path,
			Name:    name,
			SQL:     string(b),
		})
	}

	sort.Slice(migs, func(i, j int) bool { return migs[i].Version < migs[j].Version })
	return migs, nil
}

func parseVersion(filename string) (int64, bool) {
	parts := strings.SplitN(filename, "_", 2)
	if len(parts) < 2 {
		return 0, false
	}
	v, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func advisoryLockID(key string) int64 {
	sum := sha1.Sum([]byte(key))
	return int64(binary.BigEndian.Uint64(sum[:8]))
}
