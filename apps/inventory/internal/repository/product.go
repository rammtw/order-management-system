package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rammtw/order-management-system/apps/inventory/internal/model"
)

type ProductRepository struct {
	pool *pgxpool.Pool
}

func NewProductRepository(pool *pgxpool.Pool) *ProductRepository {
	return &ProductRepository{pool: pool}
}

func (r *ProductRepository) Create(ctx context.Context, sku, name, description string, priceCents int64, quantity int32) (*model.Product, error) {
	id := uuid.New().String()
	now := time.Now().UTC()

	_, err := r.pool.Exec(ctx,
		`INSERT INTO products (id, sku, name, description, price_cents, quantity, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		id, sku, name, description, priceCents, quantity, now, now,
	)
	if err != nil {
		return nil, err
	}

	return r.GetByID(ctx, id)
}

func (r *ProductRepository) GetByID(ctx context.Context, id string) (*model.Product, error) {
	return r.scanOne(r.pool.QueryRow(ctx,
		`SELECT id, sku, name, description, price_cents, quantity, reserved_quantity, created_at, updated_at
		 FROM products WHERE id = $1`, id,
	))
}

func (r *ProductRepository) GetBySKU(ctx context.Context, sku string) (*model.Product, error) {
	return r.scanOne(r.pool.QueryRow(ctx,
		`SELECT id, sku, name, description, price_cents, quantity, reserved_quantity, created_at, updated_at
		 FROM products WHERE sku = $1`, sku,
	))
}

func (r *ProductRepository) List(ctx context.Context, limit, offset int32) ([]*model.Product, int32, error) {
	var total int32
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM products`).Scan(&total); err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 20
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, sku, name, description, price_cents, quantity, reserved_quantity, created_at, updated_at
		 FROM products ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var products []*model.Product
	for rows.Next() {
		p, err := r.scanRow(rows)
		if err != nil {
			return nil, 0, err
		}
		products = append(products, p)
	}

	return products, total, rows.Err()
}

func (r *ProductRepository) UpdateStock(ctx context.Context, id string, delta int32) (*model.Product, error) {
	now := time.Now().UTC()

	_, err := r.pool.Exec(ctx,
		`UPDATE products SET quantity = quantity + $1, updated_at = $2 WHERE id = $3`,
		delta, now, id,
	)
	if err != nil {
		return nil, err
	}

	return r.GetByID(ctx, id)
}

func (r *ProductRepository) ReserveStock(ctx context.Context, orderID string, items []model.ReservationItem) ([]string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var unavailable []string
	now := time.Now().UTC()

	for _, item := range items {
		var available int32
		err := tx.QueryRow(ctx,
			`SELECT quantity - reserved_quantity FROM products WHERE id = $1 FOR UPDATE`,
			item.ProductID,
		).Scan(&available)
		if err != nil {
			return nil, err
		}

		if available < item.Quantity {
			unavailable = append(unavailable, item.ProductID)
		}
	}

	if len(unavailable) > 0 {
		return unavailable, nil
	}

	for _, item := range items {
		_, err := tx.Exec(ctx,
			`UPDATE products SET reserved_quantity = reserved_quantity + $1, updated_at = $2 WHERE id = $3`,
			item.Quantity, now, item.ProductID,
		)
		if err != nil {
			return nil, err
		}

		_, err = tx.Exec(ctx,
			`INSERT INTO stock_reservations (order_id, product_id, quantity, created_at)
			 VALUES ($1, $2, $3, $4)
			 ON CONFLICT (order_id, product_id) DO NOTHING`,
			orderID, item.ProductID, item.Quantity, now,
		)
		if err != nil {
			return nil, err
		}
	}

	return nil, tx.Commit(ctx)
}

func (r *ProductRepository) ReleaseStock(ctx context.Context, orderID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx,
		`DELETE FROM stock_reservations WHERE order_id = $1 RETURNING product_id, quantity`,
		orderID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	type release struct {
		productID string
		quantity  int32
	}

	var releases []release
	for rows.Next() {
		var rel release
		if err := rows.Scan(&rel.productID, &rel.quantity); err != nil {
			return err
		}
		releases = append(releases, rel)
	}
	rows.Close()

	if err := rows.Err(); err != nil {
		return err
	}

	now := time.Now().UTC()
	for _, rel := range releases {
		_, err := tx.Exec(ctx,
			`UPDATE products SET reserved_quantity = reserved_quantity - $1, updated_at = $2 WHERE id = $3`,
			rel.quantity, now, rel.productID,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *ProductRepository) ConfirmStock(ctx context.Context, orderID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx,
		`DELETE FROM stock_reservations WHERE order_id = $1 RETURNING product_id, quantity`,
		orderID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	type deduct struct {
		productID string
		quantity  int32
	}

	var deducts []deduct
	for rows.Next() {
		var d deduct
		if err := rows.Scan(&d.productID, &d.quantity); err != nil {
			return err
		}
		deducts = append(deducts, d)
	}
	rows.Close()

	if err := rows.Err(); err != nil {
		return err
	}

	now := time.Now().UTC()
	for _, d := range deducts {
		_, err := tx.Exec(ctx,
			`UPDATE products
			 SET quantity = quantity - $1, reserved_quantity = reserved_quantity - $1, updated_at = $2
			 WHERE id = $3`,
			d.quantity, now, d.productID,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *ProductRepository) scanOne(row pgx.Row) (*model.Product, error) {
	var p model.Product
	if err := row.Scan(
		&p.ID, &p.SKU, &p.Name, &p.Description,
		&p.PriceCents, &p.Quantity, &p.ReservedQuantity,
		&p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProductRepository) scanRow(rows pgx.Rows) (*model.Product, error) {
	var p model.Product
	if err := rows.Scan(
		&p.ID, &p.SKU, &p.Name, &p.Description,
		&p.PriceCents, &p.Quantity, &p.ReservedQuantity,
		&p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &p, nil
}
