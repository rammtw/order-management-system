package service

import (
	"context"
	"errors"

	"github.com/rammtw/order-management-system/apps/inventory/internal/model"
	"github.com/rammtw/order-management-system/apps/inventory/internal/repository"
)

var (
	ErrProductNotFound     = errors.New("product not found")
	ErrInsufficientStock   = errors.New("insufficient stock")
	ErrInvalidQuantity     = errors.New("quantity must be positive")
	ErrDuplicateSKU        = errors.New("product with this SKU already exists")
	ErrReservationNotFound = errors.New("reservation not found")
)

type InventoryService struct {
	repo *repository.ProductRepository
}

func NewInventoryService(repo *repository.ProductRepository) *InventoryService {
	return &InventoryService{repo: repo}
}

func (s *InventoryService) CreateProduct(ctx context.Context, sku, name, description string, priceCents int64, quantity int32) (*model.Product, error) {
	if quantity < 0 {
		return nil, ErrInvalidQuantity
	}
	return s.repo.Create(ctx, sku, name, description, priceCents, quantity)
}

func (s *InventoryService) GetProduct(ctx context.Context, id string) (*model.Product, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *InventoryService) GetProductBySKU(ctx context.Context, sku string) (*model.Product, error) {
	return s.repo.GetBySKU(ctx, sku)
}

func (s *InventoryService) ListProducts(ctx context.Context, limit, offset int32) ([]*model.Product, int32, error) {
	return s.repo.List(ctx, limit, offset)
}

func (s *InventoryService) UpdateStock(ctx context.Context, id string, delta int32) (*model.Product, error) {
	return s.repo.UpdateStock(ctx, id, delta)
}

func (s *InventoryService) CheckAvailability(ctx context.Context, items []model.ReservationItem) (bool, []string) {
	var unavailable []string
	for _, item := range items {
		p, err := s.repo.GetByID(ctx, item.ProductID)
		if err != nil || p.Available() < item.Quantity {
			unavailable = append(unavailable, item.ProductID)
		}
	}
	return len(unavailable) == 0, unavailable
}

func (s *InventoryService) ReserveStock(ctx context.Context, orderID string, items []model.ReservationItem) (bool, []string, error) {
	unavailable, err := s.repo.ReserveStock(ctx, orderID, items)
	if err != nil {
		return false, nil, err
	}
	if len(unavailable) > 0 {
		return false, unavailable, nil
	}
	return true, nil, nil
}

func (s *InventoryService) ReleaseStock(ctx context.Context, orderID string) error {
	return s.repo.ReleaseStock(ctx, orderID)
}

func (s *InventoryService) ConfirmStock(ctx context.Context, orderID string) error {
	return s.repo.ConfirmStock(ctx, orderID)
}
