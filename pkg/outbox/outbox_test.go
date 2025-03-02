package outbox

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/assylzhan-a/outboxie/pkg/outbox/config"
)

// TestOutbox is an integration test for the outbox.
// It requires running PostgreSQL and NATS instances.
func TestOutbox(t *testing.T) {
	// Skip the test if PostgreSQL or NATS is not available
	dbURL := "postgres://postgres:postgres@localhost:5433/outboxie"
	natsURL := "nats://localhost:4222"

	// Connect to the database
	ctx := context.Background()
	dbPool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Skip("PostgreSQL is not available:", err)
		return
	}
	defer dbPool.Close()

	// Clean up the outbox_messages table
	_, err = dbPool.Exec(ctx, "DELETE FROM outbox_messages")
	require.NoError(t, err)

	// Create a NATS connection for subscribing
	nc, err := nats.Connect(natsURL)
	if err != nil {
		t.Skip("NATS is not available:", err)
		return
	}
	defer nc.Close()

	// Create a channel to receive the message
	msgCh := make(chan []byte, 1)

	// Subscribe to the test topic
	sub, err := nc.Subscribe("test.topic", func(msg *nats.Msg) {
		msgCh <- msg.Data
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	// Flush to ensure the subscription is processed by the server
	err = nc.Flush()
	require.NoError(t, err)

	// Create the outbox configuration
	outboxConfig := config.NewOutboxConfig(dbPool, natsURL, "test-instance").
		WithPollingInterval(100 * time.Millisecond).
		WithBatchSize(10).
		WithMaxRetries(3)

	// Create the outbox
	outboxService, err := New(outboxConfig)
	require.NoError(t, err)

	// Start the outbox processor
	err = outboxService.Start(ctx)
	require.NoError(t, err)
	defer outboxService.Stop()

	// Test message
	type TestMessage struct {
		ID    uuid.UUID `json:"id"`
		Key   string    `json:"key"`
		Value string    `json:"value"`
	}

	testMsg := TestMessage{
		ID:    uuid.New(),
		Key:   "test-key",
		Value: "test-value",
	}

	// Start a transaction
	tx, err := dbPool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	// Enqueue the message
	err = outboxService.EnqueueMessage(ctx, tx, "test.topic", testMsg)
	require.NoError(t, err)

	// Commit the transaction
	err = tx.Commit(ctx)
	require.NoError(t, err)

	// Wait for the message to be received
	select {
	case receivedData := <-msgCh:
		// Unmarshal the received message
		var receivedMsg TestMessage
		err := json.Unmarshal(receivedData, &receivedMsg)
		require.NoError(t, err)

		// Verify the message
		assert.Equal(t, testMsg.ID, receivedMsg.ID)
		assert.Equal(t, testMsg.Key, receivedMsg.Key)
		assert.Equal(t, testMsg.Value, receivedMsg.Value)
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for message")
	}

	// Verify the message was marked as completed
	var status string
	err = dbPool.QueryRow(ctx, "SELECT status FROM outbox_messages WHERE payload->>'id' = $1", testMsg.ID.String()).Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, "completed", status)
}
