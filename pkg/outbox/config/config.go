package config

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type OutboxConfig struct {
	DB              *pgxpool.Pool   // connection pool
	NatsURL         string          // URL NATS
	InstanceID      string          // Unique identifier for this instance
	ProcessorConfig ProcessorConfig // Configuration for the message processor
}

type ProcessorConfig struct {
	PollingInterval time.Duration // How often to poll for new messages
	BatchSize       int           // Max number of messages to process in a batch
	MaxRetries      int           // Max retries for a failed message
}

func DefaultProcessorConfig() ProcessorConfig {
	return ProcessorConfig{
		PollingInterval: 100 * time.Millisecond,
		BatchSize:       10,
		MaxRetries:      3,
	}
}

func NewOutboxConfig(db *pgxpool.Pool, natsURL string, instanceID string) OutboxConfig {
	return OutboxConfig{
		DB:              db,
		NatsURL:         natsURL,
		InstanceID:      instanceID,
		ProcessorConfig: DefaultProcessorConfig(),
	}
}

func (c OutboxConfig) WithPollingInterval(interval time.Duration) OutboxConfig {
	c.ProcessorConfig.PollingInterval = interval
	return c
}

func (c OutboxConfig) WithBatchSize(batchSize int) OutboxConfig {
	c.ProcessorConfig.BatchSize = batchSize
	return c
}

func (c OutboxConfig) WithMaxRetries(maxRetries int) OutboxConfig {
	c.ProcessorConfig.MaxRetries = maxRetries
	return c
}
