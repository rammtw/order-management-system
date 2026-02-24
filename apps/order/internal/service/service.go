package service

import (
	"context"
	"errors"

	"github.com/rammtw/order-management-system/apps/order/internal/kafka"
	"github.com/rammtw/order-management-system/apps/order/internal/model"
	"github.com/rammtw/order-management-system/apps/order/internal/repository"
)

var (
	ErrOrderNotFound    = errors.New("order not found")
	ErrInvalidStatus    = errors.New("invalid status transition")
	ErrEmptyItems       = errors.New("order must have at least one item")
	ErrAlreadyCancelled = errors.New("order already cancelled")
)

type OrderService struct {
	repo     *repository.OrderRepository
	producer *kafka.Producer
}

func NewOrderService(repo *repository.OrderRepository, producer *kafka.Producer) *OrderService {
	return &OrderService{repo: repo, producer: producer}
}

func (s *OrderService) CreateOrder(ctx context.Context, customerID string, items []model.OrderItem) (*model.Order, error) {
	if len(items) == 0 {
		return nil, ErrEmptyItems
	}

	order, err := s.repo.Create(ctx, customerID, items)
	if err != nil {
		return nil, err
	}

	_ = s.producer.PublishOrderEvent(ctx, kafka.TopicOrderCreated, order)

	return order, nil
}

func (s *OrderService) GetOrder(ctx context.Context, id string) (*model.Order, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *OrderService) ListOrders(ctx context.Context, customerID string, limit, offset int32) ([]*model.Order, int32, error) {
	return s.repo.List(ctx, customerID, limit, offset)
}

func (s *OrderService) CancelOrder(ctx context.Context, id string) (*model.Order, error) {
	order, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if order.Status == model.OrderStatusCancelled {
		return nil, ErrAlreadyCancelled
	}

	if order.Status >= model.OrderStatusShipped {
		return nil, ErrInvalidStatus
	}

	order, err = s.repo.UpdateStatus(ctx, id, model.OrderStatusCancelled)
	if err != nil {
		return nil, err
	}

	_ = s.producer.PublishOrderEvent(ctx, kafka.TopicOrderCancelled, order)

	return order, nil
}

func (s *OrderService) UpdateOrderStatus(ctx context.Context, id string, status model.OrderStatus) (*model.Order, error) {
	order, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if order.Status == model.OrderStatusCancelled {
		return nil, ErrAlreadyCancelled
	}

	order, err = s.repo.UpdateStatus(ctx, id, status)
	if err != nil {
		return nil, err
	}

	_ = s.producer.PublishOrderEvent(ctx, kafka.TopicOrderStatus, order)

	return order, nil
}
