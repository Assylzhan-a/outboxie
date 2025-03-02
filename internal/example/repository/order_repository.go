package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/assylzhan-a/outboxie/internal/example/model"
)

type OrderRepository interface {
	CreateOrder(ctx context.Context, tx pgx.Tx, order *model.Order) error
}

type PostgresOrderRepository struct {
	db *pgxpool.Pool
}

func NewPostgresOrderRepository(db *pgxpool.Pool) *PostgresOrderRepository {
	return &PostgresOrderRepository{
		db: db,
	}
}

func (r *PostgresOrderRepository) CreateOrder(ctx context.Context, tx pgx.Tx, order *model.Order) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO orders (id, customer_id, amount, status, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, order.ID, order.CustomerID, order.Amount, order.Status, order.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to insert order: %w", err)
	}

	return nil
}
