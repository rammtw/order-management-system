package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rammtw/order-management-system/apps/order/internal/model"
)

type OrderRepository struct {
	pool *pgxpool.Pool
}

func NewOrderRepository(pool *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{pool: pool}
}

func (r *OrderRepository) Create(ctx context.Context, customerID string, items []model.OrderItem) (*model.Order, error) {
	var totalCents int64
	for _, item := range items {
		totalCents += item.PriceCents * int64(item.Quantity)
	}

	id := uuid.New().String()
	now := time.Now().UTC()

	itemsJSON, err := json.Marshal(items)
	if err != nil {
		return nil, err
	}

	_, err = r.pool.Exec(ctx,
		`INSERT INTO orders (id, customer_id, items, status, total_cents, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		id, customerID, itemsJSON, model.OrderStatusPending, totalCents, now, now,
	)
	if err != nil {
		return nil, err
	}

	return &model.Order{
		ID:         id,
		CustomerID: customerID,
		Items:      items,
		Status:     model.OrderStatusPending,
		TotalCents: totalCents,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

func (r *OrderRepository) GetByID(ctx context.Context, id string) (*model.Order, error) {
	var (
		o         model.Order
		itemsJSON []byte
	)

	err := r.pool.QueryRow(ctx,
		`SELECT id, customer_id, items, status, total_cents, created_at, updated_at
		 FROM orders WHERE id = $1`, id,
	).Scan(&o.ID, &o.CustomerID, &itemsJSON, &o.Status, &o.TotalCents, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(itemsJSON, &o.Items); err != nil {
		return nil, err
	}

	return &o, nil
}

func (r *OrderRepository) List(ctx context.Context, customerID string, limit, offset int32) ([]*model.Order, int32, error) {
	var total int32
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM orders WHERE customer_id = $1`, customerID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 20
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, customer_id, items, status, total_cents, created_at, updated_at
		 FROM orders WHERE customer_id = $1
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		customerID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var orders []*model.Order
	for rows.Next() {
		var (
			o         model.Order
			itemsJSON []byte
		)
		if err := rows.Scan(&o.ID, &o.CustomerID, &itemsJSON, &o.Status, &o.TotalCents, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, 0, err
		}
		if err := json.Unmarshal(itemsJSON, &o.Items); err != nil {
			return nil, 0, err
		}
		orders = append(orders, &o)
	}

	return orders, total, rows.Err()
}

func (r *OrderRepository) UpdateStatus(ctx context.Context, id string, status model.OrderStatus) (*model.Order, error) {
	now := time.Now().UTC()

	_, err := r.pool.Exec(ctx,
		`UPDATE orders SET status = $1, updated_at = $2 WHERE id = $3`,
		status, now, id,
	)
	if err != nil {
		return nil, err
	}

	return r.GetByID(ctx, id)
}
