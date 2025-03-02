package repository

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/assylzhan-a/outboxie/pkg/outbox/model"
)

// TestPostgresRepository is an integration test for the PostgreSQL repository.
// It requires a running PostgreSQL instance.
// To run this test, set the environment variable OUTBOXIE_TEST_DB_URL to a PostgreSQL connection string.
// Example: OUTBOXIE_TEST_DB_URL=postgres://postgres:postgres@localhost:5433/outboxie
func TestPostgresRepository(t *testing.T) {
	// Skip the test if the environment variable is not set
	dbURL := "postgres://postgres:postgres@localhost:5433/outboxie"

	// Connect to the database
	ctx := context.Background()
	dbPool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)
	defer dbPool.Close()

	// Create a repository
	repo := NewPostgresRepository(dbPool)

	// Clean up the outbox_messages table
	_, err = dbPool.Exec(ctx, "DELETE FROM outbox_messages")
	require.NoError(t, err)

	// Create a test message
	message, err := model.NewOutboxMessage("test.topic", map[string]interface{}{
		"key": "value",
	})
	require.NoError(t, err)

	// Start a transaction
	tx, err := dbPool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	// Test EnqueueMessage
	t.Run("EnqueueMessage", func(t *testing.T) {
		err := repo.EnqueueMessage(ctx, tx, message)
		assert.NoError(t, err)
	})

	// Commit the transaction
	err = tx.Commit(ctx)
	require.NoError(t, err)

	// Test GetPendingMessages
	t.Run("GetPendingMessages", func(t *testing.T) {
		messages, err := repo.GetPendingMessages(ctx, 10)
		assert.NoError(t, err)
		assert.Len(t, messages, 1)
		assert.Equal(t, message.ID, messages[0].ID)
		assert.Equal(t, message.Topic, messages[0].Topic)
		assert.Equal(t, model.StatusPending, messages[0].Status)
	})

	// Test MarkMessageAsProcessing
	t.Run("MarkMessageAsProcessing", func(t *testing.T) {
		err := repo.MarkMessageAsProcessing(ctx, message.ID)
		assert.NoError(t, err)

		// Verify the message status
		var status string
		err = dbPool.QueryRow(ctx, "SELECT status FROM outbox_messages WHERE id = $1", message.ID).Scan(&status)
		assert.NoError(t, err)
		assert.Equal(t, string(model.StatusProcessing), status)
	})

	// Test MarkMessageAsCompleted
	t.Run("MarkMessageAsCompleted", func(t *testing.T) {
		err := repo.MarkMessageAsCompleted(ctx, message.ID)
		assert.NoError(t, err)

		// Verify the message status
		var status string
		var processedAt time.Time
		err = dbPool.QueryRow(ctx, "SELECT status, processed_at FROM outbox_messages WHERE id = $1", message.ID).Scan(&status, &processedAt)
		assert.NoError(t, err)
		assert.Equal(t, string(model.StatusCompleted), status)
		assert.False(t, processedAt.IsZero())
	})

	// Test MarkMessageAsFailed
	t.Run("MarkMessageAsFailed", func(t *testing.T) {
		// Create a new message for this test
		message2, err := model.NewOutboxMessage("test.topic", map[string]interface{}{
			"key": "value2",
		})
		require.NoError(t, err)

		// Start a transaction
		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		defer tx.Rollback(ctx)

		// Enqueue the message
		err = repo.EnqueueMessage(ctx, tx, message2)
		require.NoError(t, err)

		// Commit the transaction
		err = tx.Commit(ctx)
		require.NoError(t, err)

		// Mark the message as processing
		err = repo.MarkMessageAsProcessing(ctx, message2.ID)
		require.NoError(t, err)

		// Mark the message as failed
		testErr := assert.AnError
		err = repo.MarkMessageAsFailed(ctx, message2.ID, testErr)
		assert.NoError(t, err)

		// Verify the message status
		var status string
		var retryCount int
		var errorMsg *string
		err = dbPool.QueryRow(ctx, "SELECT status, retry_count, error FROM outbox_messages WHERE id = $1", message2.ID).Scan(&status, &retryCount, &errorMsg)
		assert.NoError(t, err)
		assert.Equal(t, string(model.StatusFailed), status)
		assert.Equal(t, 1, retryCount)
		assert.NotNil(t, errorMsg)
		assert.Equal(t, testErr.Error(), *errorMsg)
	})
}
