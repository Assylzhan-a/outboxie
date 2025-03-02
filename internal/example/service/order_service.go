package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/assylzhan-a/outboxie/internal/example/model"
	"github.com/assylzhan-a/outboxie/internal/example/repository"
	"github.com/assylzhan-a/outboxie/pkg/outbox"
)

type OrderService struct {
	db            *pgxpool.Pool
	orderRepo     repository.OrderRepository
	outboxService *outbox.Outbox
}

func NewOrderService(db *pgxpool.Pool, orderRepo repository.OrderRepository, outboxService *outbox.Outbox) *OrderService {
	return &OrderService{
		db:            db,
		orderRepo:     orderRepo,
		outboxService: outboxService,
	}
}

func (s *OrderService) CreateOrder(ctx context.Context, orderData *model.Order) (*model.Order, error) {
	if orderData.ID == uuid.Nil {
		orderData.ID = uuid.New()
	}

	if orderData.CreatedAt.IsZero() {
		orderData.CreatedAt = time.Now().UTC()
	}

	if orderData.Status == "" {
		orderData.Status = "pending"
	}

	// Start a transaction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Create the order in the database
	if err := s.orderRepo.CreateOrder(ctx, tx, orderData); err != nil {
		return nil, err
	}

	// Create an order created event
	event := model.OrderCreatedEvent{
		OrderID:    orderData.ID,
		CustomerID: orderData.CustomerID,
		Amount:     orderData.Amount,
		CreatedAt:  orderData.CreatedAt,
	}

	// Enqueue the event to be published after the transaction is committed
	if err := s.outboxService.EnqueueMessage(ctx, tx, "orders.created", event); err != nil {
		return nil, fmt.Errorf("failed to enqueue message: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return orderData, nil
}
