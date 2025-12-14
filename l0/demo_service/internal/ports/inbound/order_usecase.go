package inbound

import (
	"context"
	"demo_service/internal/core/domain"
)

type OrderUseCase interface {
	GetByID(ctx context.Context, orderUID string) (domain.Order, error)
	Ingest(ctx context.Context, order domain.Order) error
	WarmCache(ctx context.Context, limit int) (int, error)
	ListPage(ctx context.Context, page, pageSize int) (orders []domain.Order, total int, err error)
}
