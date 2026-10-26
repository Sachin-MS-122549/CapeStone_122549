package repository

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"order-management/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("order not found")

type OrderRepository interface {
	Create(context.Context, model.Order) (model.Order, error)
	Get(context.Context, int64) (model.Order, error)
	List(context.Context, int, int) ([]model.Order, error)
	UpdateStatus(context.Context, int64, model.OrderStatus) (model.Order, error)
	Ping(context.Context) error
}

type PostgresRepository struct{ db *pgxpool.Pool }

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository { return &PostgresRepository{db: db} }
func (r *PostgresRepository) Ping(ctx context.Context) error     { return r.db.Ping(ctx) }
func (r *PostgresRepository) Create(ctx context.Context, o model.Order) (model.Order, error) {
	err := r.db.QueryRow(ctx, `INSERT INTO orders(customer_id,product_id,quantity,unit_price,total_amount,status) VALUES($1,$2,$3,$4,$5,$6) RETURNING id,created_at,updated_at`,
		o.CustomerID, o.ProductID, o.Quantity, o.UnitPrice, o.TotalAmount, o.Status).Scan(&o.ID, &o.CreatedAt, &o.UpdatedAt)
	return o, err
}

const orderColumns = `id,customer_id,product_id,quantity,unit_price,total_amount,status,created_at,updated_at`

func scanOrder(row interface{ Scan(...any) error }) (model.Order, error) {
	var o model.Order
	err := row.Scan(&o.ID, &o.CustomerID, &o.ProductID, &o.Quantity, &o.UnitPrice, &o.TotalAmount, &o.Status, &o.CreatedAt, &o.UpdatedAt)
	return o, err
}
func (r *PostgresRepository) Get(ctx context.Context, id int64) (model.Order, error) {
	o, err := scanOrder(r.db.QueryRow(ctx, `SELECT `+orderColumns+` FROM orders WHERE id=$1`, id))
	if err != nil {
		if err.Error() == "no rows in result set" {
			return o, ErrNotFound
		}
		return o, err
	}
	return o, nil
}
func (r *PostgresRepository) List(ctx context.Context, limit, offset int) ([]model.Order, error) {
	rows, err := r.db.Query(ctx, `SELECT `+orderColumns+` FROM orders ORDER BY id DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.Order, 0)
	for rows.Next() {
		o, e := scanOrder(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, o)
	}
	return out, rows.Err()
}
func (r *PostgresRepository) UpdateStatus(ctx context.Context, id int64, s model.OrderStatus) (model.Order, error) {
	o, err := scanOrder(r.db.QueryRow(ctx, `UPDATE orders SET status=$1,updated_at=NOW() WHERE id=$2 RETURNING `+orderColumns, s, id))
	if err != nil {
		if err.Error() == "no rows in result set" {
			return o, ErrNotFound
		}
		return o, err
	}
	return o, nil
}

// MemoryRepository makes the project usable locally without PostgreSQL. Use PostgreSQL for persistence.
type MemoryRepository struct {
	mu     sync.RWMutex
	orders map[int64]model.Order
	nextID int64
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{orders: map[int64]model.Order{}, nextID: 1}
}
func (r *MemoryRepository) Ping(context.Context) error { return nil }
func (r *MemoryRepository) Create(_ context.Context, o model.Order) (model.Order, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	o.ID = r.nextID
	r.nextID++
	o.CreatedAt = now
	o.UpdatedAt = now
	r.orders[o.ID] = o
	return o, nil
}
func (r *MemoryRepository) Get(_ context.Context, id int64) (model.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	o, ok := r.orders[id]
	if !ok {
		return o, ErrNotFound
	}
	return o, nil
}
func (r *MemoryRepository) List(_ context.Context, limit, offset int) ([]model.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]int64, 0, len(r.orders))
	for id := range r.orders {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] > ids[j] })
	out := make([]model.Order, 0)
	for i := offset; i < len(ids) && len(out) < limit; i++ {
		out = append(out, r.orders[ids[i]])
	}
	return out, nil
}
func (r *MemoryRepository) UpdateStatus(_ context.Context, id int64, s model.OrderStatus) (model.Order, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	o, ok := r.orders[id]
	if !ok {
		return o, ErrNotFound
	}
	o.Status = s
	o.UpdatedAt = time.Now().UTC()
	r.orders[id] = o
	return o, nil
}
