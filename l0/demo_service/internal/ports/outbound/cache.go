package outbound

import (
	"context"

	"demo_service/internal/core/domain"
)

type OrderCache interface {
	Get(ctx context.Context, orderUID string) (domain.Order, bool)
	Set(ctx context.Context, order domain.Order)
	BulkSet(ctx context.Context, orders []domain.Order)
	Len(ctx context.Context) int
}
