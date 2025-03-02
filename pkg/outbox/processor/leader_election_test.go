package processor

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabaseLeaderElection(t *testing.T) {
	dbURL := "postgres://postgres:postgres@localhost:5433/outboxie"

	ctx := context.Background()
	dbPool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Skip("PostgreSQL is not available:", err)
		return
	}
	defer dbPool.Close()

	// Clean up the leader_election table
	_, err = dbPool.Exec(ctx, "DELETE FROM leader_election")
	require.NoError(t, err)

	// Create a leader election instance
	instanceID := "test-instance"
	le := NewDatabaseLeaderElection(dbPool, instanceID)

	// Test initial state
	assert.False(t, le.IsLeader(), "Should not be the leader initially")

	// Test Start
	err = le.Start(ctx)
	assert.NoError(t, err)

	// Give it a moment to claim leadership
	time.Sleep(100 * time.Millisecond)

	// Should be the leader now
	assert.True(t, le.IsLeader(), "Should be the leader after Start")

	// Test Stop
	err = le.Stop()
	assert.NoError(t, err)
	assert.False(t, le.IsLeader(), "Should not be the leader after Stop")

	// Verify the leader record was removed
	var count int
	err = dbPool.QueryRow(ctx, "SELECT COUNT(*) FROM leader_election WHERE instance_id = $1", instanceID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 0, count, "Leader record should be removed after Stop")
}
