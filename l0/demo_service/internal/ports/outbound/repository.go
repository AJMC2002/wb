package outbound

import (
	"context"

	"demo_service/internal/core/domain"
)

type OrderRepository interface {
	Upsert(ctx context.Context, order domain.Order) error
	GetByID(ctx context.Context, orderUID string) (domain.Order, error)
	ListLatest(ctx context.Context, limit int) ([]domain.Order, error)
	ListOrderUIDs(ctx context.Context, limit, offset int) ([]string, error)
	CountOrders(ctx context.Context) (int, error)
}
