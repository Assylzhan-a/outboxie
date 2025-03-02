package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/assylzhan-a/outboxie/pkg/outbox/model"
)

type Repository interface {
	EnqueueMessage(ctx context.Context, tx pgx.Tx, message *model.OutboxMessage) error

	GetPendingMessages(ctx context.Context, limit int) ([]*model.OutboxMessage, error)

	MarkMessageAsProcessing(ctx context.Context, id uuid.UUID) error

	MarkMessageAsCompleted(ctx context.Context, id uuid.UUID) error

	MarkMessageAsFailed(ctx context.Context, id uuid.UUID, err error) error
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{
		db: db,
	}
}

// EnqueueMessage stores a message in the outbox as part of a transaction
func (r *PostgresRepository) EnqueueMessage(ctx context.Context, tx pgx.Tx, message *model.OutboxMessage) error {
	query := `
		INSERT INTO outbox_messages (
			id, topic, payload, created_at, status
		) VALUES (
			$1, $2, $3, $4, $5
		)
	`

	_, err := tx.Exec(ctx, query,
		message.ID,
		message.Topic,
		message.Payload,
		message.CreatedAt,
		message.Status,
	)

	if err != nil {
		return fmt.Errorf("failed to enqueue message: %w", err)
	}

	return nil
}

// GetPendingMessages retrieves messages that need processing
func (r *PostgresRepository) GetPendingMessages(ctx context.Context, limit int) ([]*model.OutboxMessage, error) {
	query := `
		SELECT 
			id, topic, payload, created_at, processed_at, status, retry_count, error, sequence_number
		FROM 
			outbox_messages
		WHERE 
			status = $1
		ORDER BY 
			sequence_number ASC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, model.StatusPending, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending messages: %w", err)
	}
	defer rows.Close()

	var messages []*model.OutboxMessage
	for rows.Next() {
		var msg model.OutboxMessage
		err := rows.Scan(
			&msg.ID,
			&msg.Topic,
			&msg.Payload,
			&msg.CreatedAt,
			&msg.ProcessedAt,
			&msg.Status,
			&msg.RetryCount,
			&msg.Error,
			&msg.SequenceNumber,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, &msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over messages: %w", err)
	}

	return messages, nil
}

// MarkMessageAsProcessing updates a message to processing status
func (r *PostgresRepository) MarkMessageAsProcessing(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE outbox_messages
		SET status = $1
		WHERE id = $2 AND status = $3
	`

	result, err := r.db.Exec(ctx, query, model.StatusProcessing, id, model.StatusPending)
	if err != nil {
		return fmt.Errorf("failed to mark message as processing: %w", err)
	}

	if result.RowsAffected() == 0 {
		return errors.New("message not found or already being processed")
	}

	return nil
}

// MarkMessageAsCompleted updates a message to completed status
func (r *PostgresRepository) MarkMessageAsCompleted(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	query := `
		UPDATE outbox_messages
		SET status = $1, processed_at = $2
		WHERE id = $3
	`

	result, err := r.db.Exec(ctx, query, model.StatusCompleted, now, id)
	if err != nil {
		return fmt.Errorf("failed to mark message as completed: %w", err)
	}

	if result.RowsAffected() == 0 {
		return errors.New("message not found")
	}

	return nil
}

// MarkMessageAsFailed updates a message to failed status and increments retry count
func (r *PostgresRepository) MarkMessageAsFailed(ctx context.Context, id uuid.UUID, err error) error {
	query := `
		UPDATE outbox_messages
		SET status = $1, retry_count = retry_count + 1, error = $2
		WHERE id = $3
	`

	var errorMsg *string
	if err != nil {
		errStr := err.Error()
		errorMsg = &errStr
	}

	result, err := r.db.Exec(ctx, query, model.StatusFailed, errorMsg, id)
	if err != nil {
		return fmt.Errorf("failed to mark message as failed: %w", err)
	}

	if result.RowsAffected() == 0 {
		return errors.New("message not found")
	}

	return nil
}
