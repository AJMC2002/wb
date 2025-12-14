package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr      string
	DatabaseURL   string
	MigrationsDir string

	KafkaBrokers       []string
	KafkaTopic         string
	KafkaConsumerGroup string

	CacheWarmLimit int
	KafkaMaxBytes  int
	KafkaMinBytes  int

	ShutdownTimeout time.Duration
}

func Load() (Config, error) {
	var c Config

	c.HTTPAddr = getenv("APP_HTTP_ADDR", ":8081")
	c.DatabaseURL = os.Getenv("DATABASE_URL")
	if c.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}

	c.MigrationsDir = getenv("MIGRATIONS_DIR", "internal/migrations")

	brokers := strings.TrimSpace(os.Getenv("KAFKA_BROKERS"))
	if brokers == "" {
		return Config{}, errors.New("KAFKA_BROKERS is required")
	}
	c.KafkaBrokers = splitCSV(brokers)

	c.KafkaTopic = getenv("KAFKA_TOPIC", "orders")
	c.KafkaConsumerGroup = getenv("KAFKA_CONSUMER_GROUP", "orders-service")

	c.CacheWarmLimit = getenvInt("CACHE_WARM_LIMIT", 100)
	c.KafkaMinBytes = getenvInt("KAFKA_MIN_BYTES", 1e3)
	c.KafkaMaxBytes = getenvInt("KAFKA_MAX_BYTES", 10e6)

	c.ShutdownTimeout = getenvDuration("SHUTDOWN_TIMEOUT", 10*time.Second)

	return c, nil
}

func getenv(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}

func getenvInt(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func getenvDuration(key string, def time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
