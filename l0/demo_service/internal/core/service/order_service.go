package service

import (
	"context"
	"errors"
	"fmt"

	"demo_service/internal/core/domain"
	"demo_service/internal/ports/inbound"
	"demo_service/internal/ports/outbound"
)

type OrderService struct {
	repo  outbound.OrderRepository
	cache outbound.OrderCache
}

func NewOrderService(repo outbound.OrderRepository, cache outbound.OrderCache) *OrderService {
	return &OrderService{repo: repo, cache: cache}
}

func (s *OrderService) Ingest(ctx context.Context, order domain.Order) error {
	if err := order.Validate(); err != nil {
		return fmt.Errorf("validate: %w", err)
	}

	if err := s.repo.Upsert(ctx, order); err != nil {
		return fmt.Errorf("db upsert: %w", err)
	}

	s.cache.Set(ctx, order)
	return nil
}

func (s *OrderService) GetByID(ctx context.Context, orderUID string) (domain.Order, error) {
	if orderUID == "" {
		return domain.Order{}, domain.ErrNotFound
	}

	if o, ok := s.cache.Get(ctx, orderUID); ok {
		return o, nil
	}

	o, err := s.repo.GetByID(ctx, orderUID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.Order{}, domain.ErrNotFound
		}
		return domain.Order{}, fmt.Errorf("db get: %w", err)
	}

	s.cache.Set(ctx, o)
	return o, nil
}

func (s *OrderService) WarmCache(ctx context.Context, limit int) (int, error) {
	if limit <= 0 {
		return 0, nil
	}

	orders, err := s.repo.ListLatest(ctx, limit)
	if err != nil {
		return 0, fmt.Errorf("db list latest: %w", err)
	}

	s.cache.BulkSet(ctx, orders)
	return len(orders), nil
}

func (s *OrderService) ListPage(ctx context.Context, page, pageSize int) ([]domain.Order, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	total, err := s.repo.CountOrders(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("db count: %w", err)
	}
	if total == 0 {
		return []domain.Order{}, 0, nil
	}

	uids, err := s.repo.ListOrderUIDs(ctx, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("db list uids: %w", err)
	}

	orders := make([]domain.Order, 0, len(uids))
	for _, uid := range uids {
		o, err := s.repo.GetByID(ctx, uid)
		if err != nil {
			continue
		}
		orders = append(orders, o)
	}
	return orders, total, nil
}

var _ inbound.OrderUseCase = (*OrderService)(nil)
