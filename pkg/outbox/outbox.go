package outbox

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/assylzhan-a/outboxie/pkg/outbox/config"
	"github.com/assylzhan-a/outboxie/pkg/outbox/model"
	"github.com/assylzhan-a/outboxie/pkg/outbox/processor"
	"github.com/assylzhan-a/outboxie/pkg/outbox/publisher"
	"github.com/assylzhan-a/outboxie/pkg/outbox/repository"
)

type Outbox struct {
	repo      repository.Repository
	publisher publisher.Publisher
	processor *processor.Processor
}

// New creates a new outbox instance
func New(cfg config.OutboxConfig) (*Outbox, error) {
	repo := repository.NewPostgresRepository(cfg.DB)

	pub, err := publisher.NewNatsPublisher(cfg.NatsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create NATS publisher: %w", err)
	}

	leaderElection := processor.NewDatabaseLeaderElection(cfg.DB, cfg.InstanceID)
	proc := processor.NewProcessor(repo, pub, leaderElection, cfg.ProcessorConfig)

	return &Outbox{
		repo:      repo,
		publisher: pub,
		processor: proc,
	}, nil
}

func (o *Outbox) Start(ctx context.Context) error {
	return o.processor.Start(ctx)
}

func (o *Outbox) Stop() error {
	return o.processor.Stop()
}

// EnqueueMessage stores a message to be published after transaction commit
// The message is stored in the outbox table as part of the transaction
func (o *Outbox) EnqueueMessage(ctx context.Context, tx pgx.Tx, topic string, payload interface{}) error {
	msg, err := model.NewOutboxMessage(topic, payload)
	if err != nil {
		return fmt.Errorf("failed to create outbox message: %w", err)
	}

	if err := o.repo.EnqueueMessage(ctx, tx, msg); err != nil {
		return fmt.Errorf("failed to enqueue message: %w", err)
	}

	return nil
}
