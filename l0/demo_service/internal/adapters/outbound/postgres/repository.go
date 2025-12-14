package postgres

import (
	"context"
	"errors"
	"fmt"

	"demo_service/internal/core/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderRepository struct {
	pool *pgxpool.Pool
}

func NewOrderRepository(pool *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{pool: pool}
}

func (r *OrderRepository) Upsert(ctx context.Context, order domain.Order) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// orders
	_, err = tx.Exec(ctx, `
		INSERT INTO orders (
			order_uid, track_number, entry, locale, internal_signature, customer_id,
			delivery_service, shardkey, sm_id, date_created, oof_shard, updated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11, now()
		)
		ON CONFLICT (order_uid) DO UPDATE SET
			track_number = EXCLUDED.track_number,
			entry = EXCLUDED.entry,
			locale = EXCLUDED.locale,
			internal_signature = EXCLUDED.internal_signature,
			customer_id = EXCLUDED.customer_id,
			delivery_service = EXCLUDED.delivery_service,
			shardkey = EXCLUDED.shardkey,
			sm_id = EXCLUDED.sm_id,
			date_created = EXCLUDED.date_created,
			oof_shard = EXCLUDED.oof_shard,
			updated_at = now()
	`, order.OrderUID, order.TrackNumber, order.Entry, order.Locale, order.InternalSignature,
		order.CustomerID, order.DeliveryService, order.ShardKey, order.SmID, order.DateCreated, order.OofShard)
	if err != nil {
		return fmt.Errorf("upsert orders: %w", err)
	}

	// deliveries
	d := order.Delivery
	_, err = tx.Exec(ctx, `
		INSERT INTO deliveries (order_uid, name, phone, zip, city, address, region, email)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (order_uid) DO UPDATE SET
			name=EXCLUDED.name,
			phone=EXCLUDED.phone,
			zip=EXCLUDED.zip,
			city=EXCLUDED.city,
			address=EXCLUDED.address,
			region=EXCLUDED.region,
			email=EXCLUDED.email
	`, order.OrderUID, d.Name, d.Phone, d.Zip, d.City, d.Address, d.Region, d.Email)
	if err != nil {
		return fmt.Errorf("upsert deliveries: %w", err)
	}

	// payments
	p := order.Payment
	_, err = tx.Exec(ctx, `
		INSERT INTO payments (
			order_uid, transaction, request_id, currency, provider, amount, payment_dt, bank,
			delivery_cost, goods_total, custom_fee
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (order_uid) DO UPDATE SET
			transaction=EXCLUDED.transaction,
			request_id=EXCLUDED.request_id,
			currency=EXCLUDED.currency,
			provider=EXCLUDED.provider,
			amount=EXCLUDED.amount,
			payment_dt=EXCLUDED.payment_dt,
			bank=EXCLUDED.bank,
			delivery_cost=EXCLUDED.delivery_cost,
			goods_total=EXCLUDED.goods_total,
			custom_fee=EXCLUDED.custom_fee
	`, order.OrderUID, p.Transaction, p.RequestID, p.Currency, p.Provider, p.Amount,
		p.PaymentDT, p.Bank, p.DeliveryCost, p.GoodsTotal, p.CustomFee)
	if err != nil {
		return fmt.Errorf("upsert payments: %w", err)
	}

	// items
	_, err = tx.Exec(ctx, `DELETE FROM items WHERE order_uid = $1`, order.OrderUID)
	if err != nil {
		return fmt.Errorf("delete items: %w", err)
	}

	for _, it := range order.Items {
		_, err = tx.Exec(ctx, `
			INSERT INTO items (
				order_uid, chrt_id, track_number, price, rid, name, sale, size,
				total_price, nm_id, brand, status
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		`, order.OrderUID, it.ChrtID, it.TrackNumber, it.Price, it.RID, it.Name,
			it.Sale, it.Size, it.TotalPrice, it.NmID, it.Brand, it.Status)
		if err != nil {
			return fmt.Errorf("insert item: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

func (r *OrderRepository) GetByID(ctx context.Context, orderUID string) (domain.Order, error) {
	var o domain.Order

	row := r.pool.QueryRow(ctx, `
		SELECT
			o.order_uid, o.track_number, o.entry, o.locale, o.internal_signature, o.customer_id,
			o.delivery_service, o.shardkey, o.sm_id, o.date_created, o.oof_shard,

			d.name, d.phone, d.zip, d.city, d.address, d.region, d.email,

			p.transaction, p.request_id, p.currency, p.provider, p.amount, p.payment_dt, p.bank,
			p.delivery_cost, p.goods_total, p.custom_fee
		FROM orders o
		JOIN deliveries d ON d.order_uid = o.order_uid
		JOIN payments p ON p.order_uid = o.order_uid
		WHERE o.order_uid = $1
	`, orderUID)

	err := row.Scan(
		&o.OrderUID, &o.TrackNumber, &o.Entry, &o.Locale, &o.InternalSignature, &o.CustomerID,
		&o.DeliveryService, &o.ShardKey, &o.SmID, &o.DateCreated, &o.OofShard,

		&o.Delivery.Name, &o.Delivery.Phone, &o.Delivery.Zip, &o.Delivery.City,
		&o.Delivery.Address, &o.Delivery.Region, &o.Delivery.Email,

		&o.Payment.Transaction, &o.Payment.RequestID, &o.Payment.Currency, &o.Payment.Provider,
		&o.Payment.Amount, &o.Payment.PaymentDT, &o.Payment.Bank, &o.Payment.DeliveryCost,
		&o.Payment.GoodsTotal, &o.Payment.CustomFee,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Order{}, domain.ErrNotFound
		}
		return domain.Order{}, fmt.Errorf("scan order: %w", err)
	}

	// items
	rows, err := r.pool.Query(ctx, `
		SELECT chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status
		FROM items
		WHERE order_uid = $1
		ORDER BY id ASC
	`, orderUID)
	if err != nil {
		return domain.Order{}, fmt.Errorf("query items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var it domain.Item
		if err := rows.Scan(
			&it.ChrtID, &it.TrackNumber, &it.Price, &it.RID, &it.Name,
			&it.Sale, &it.Size, &it.TotalPrice, &it.NmID, &it.Brand, &it.Status,
		); err != nil {
			return domain.Order{}, fmt.Errorf("scan item: %w", err)
		}
		o.Items = append(o.Items, it)
	}
	if err := rows.Err(); err != nil {
		return domain.Order{}, fmt.Errorf("items rows: %w", err)
	}

	return o, nil
}

func (r *OrderRepository) ListLatest(ctx context.Context, limit int) ([]domain.Order, error) {
	if limit <= 0 {
		return []domain.Order{}, nil
	}

	rows, err := r.pool.Query(ctx, `
		SELECT order_uid
		FROM orders
		ORDER BY date_created DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list latest ids: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows err: %w", err)
	}

	out := make([]domain.Order, 0, len(ids))
	for _, id := range ids {
		o, err := r.GetByID(ctx, id)
		if err != nil {
			continue
		}
		out = append(out, o)
	}

	return out, nil
}

func (r *OrderRepository) CountOrders(ctx context.Context) (int, error) {
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM orders`).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

func (r *OrderRepository) ListOrderUIDs(ctx context.Context, limit, offset int) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT order_uid
		FROM orders
		ORDER BY date_created DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		out = append(out, uid)
	}
	return out, rows.Err()
}
