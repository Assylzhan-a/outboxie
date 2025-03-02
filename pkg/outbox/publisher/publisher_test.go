package publisher

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNatsPublisher is an integration test for the NATS publisher.
// It requires a running NATS instance.
func TestNatsPublisher(t *testing.T) {
	// Skip the test if NATS is not available
	natsURL := "nats://localhost:4222"

	// Create a publisher
	pub, err := NewNatsPublisher(natsURL)
	if err != nil {
		t.Skip("NATS is not available:", err)
		return
	}
	defer pub.Close()

	// Create a NATS connection for subscribing
	nc, err := nats.Connect(natsURL)
	require.NoError(t, err)
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

	// Test message
	type TestMessage struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}

	testMsg := TestMessage{
		Key:   "test-key",
		Value: "test-value",
	}

	// Marshal the test message
	payload, err := json.Marshal(testMsg)
	require.NoError(t, err)

	// Publish the message
	ctx := context.Background()
	err = pub.Publish(ctx, "test.topic", payload)
	require.NoError(t, err)

	// Wait for the message to be received
	select {
	case receivedData := <-msgCh:
		// Unmarshal the received message
		var receivedMsg TestMessage
		err := json.Unmarshal(receivedData, &receivedMsg)
		require.NoError(t, err)

		// Verify the message
		assert.Equal(t, testMsg.Key, receivedMsg.Key)
		assert.Equal(t, testMsg.Value, receivedMsg.Value)
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for message")
	}
}
