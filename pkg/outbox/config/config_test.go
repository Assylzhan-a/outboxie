package config

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

func TestDefaultProcessorConfig(t *testing.T) {
	config := DefaultProcessorConfig()

	assert.Equal(t, 100*time.Millisecond, config.PollingInterval)
	assert.Equal(t, 10, config.BatchSize)
	assert.Equal(t, 3, config.MaxRetries)
}

func TestNewOutboxConfig(t *testing.T) {
	var db *pgxpool.Pool = nil
	natsURL := "nats://localhost:4222"
	instanceID := "test-instance"

	config := NewOutboxConfig(db, natsURL, instanceID)

	assert.Equal(t, db, config.DB)
	assert.Equal(t, natsURL, config.NatsURL)
	assert.Equal(t, instanceID, config.InstanceID)

	assert.Equal(t, 100*time.Millisecond, config.ProcessorConfig.PollingInterval)
	assert.Equal(t, 10, config.ProcessorConfig.BatchSize)
	assert.Equal(t, 3, config.ProcessorConfig.MaxRetries)
}

func TestConfigBuilderMethods(t *testing.T) {

	var db *pgxpool.Pool = nil
	natsURL := "nats://localhost:4222"
	instanceID := "test-instance"

	config := NewOutboxConfig(db, natsURL, instanceID)

	newInterval := 200 * time.Millisecond
	configWithInterval := config.WithPollingInterval(newInterval)
	assert.Equal(t, newInterval, configWithInterval.ProcessorConfig.PollingInterval)

	newBatchSize := 20
	configWithBatchSize := config.WithBatchSize(newBatchSize)
	assert.Equal(t, newBatchSize, configWithBatchSize.ProcessorConfig.BatchSize)

	newMaxRetries := 5
	configWithMaxRetries := config.WithMaxRetries(newMaxRetries)
	assert.Equal(t, newMaxRetries, configWithMaxRetries.ProcessorConfig.MaxRetries)

	chainedConfig := config.
		WithPollingInterval(newInterval).
		WithBatchSize(newBatchSize).
		WithMaxRetries(newMaxRetries)

	assert.Equal(t, newInterval, chainedConfig.ProcessorConfig.PollingInterval)
	assert.Equal(t, newBatchSize, chainedConfig.ProcessorConfig.BatchSize)
	assert.Equal(t, newMaxRetries, chainedConfig.ProcessorConfig.MaxRetries)
}
