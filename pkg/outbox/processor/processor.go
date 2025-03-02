package processor

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/assylzhan-a/outboxie/pkg/outbox/config"
	"github.com/assylzhan-a/outboxie/pkg/outbox/model"
	"github.com/assylzhan-a/outboxie/pkg/outbox/publisher"
	"github.com/assylzhan-a/outboxie/pkg/outbox/repository"
)

type Processor struct {
	repo           repository.Repository
	publisher      publisher.Publisher
	leaderElection LeaderElection
	config         config.ProcessorConfig
	stopCh         chan struct{}
	wg             sync.WaitGroup
	mu             sync.Mutex
	running        bool
}

func NewProcessor(
	repo repository.Repository,
	publisher publisher.Publisher,
	leaderElection LeaderElection,
	config config.ProcessorConfig,
) *Processor {
	return &Processor{
		repo:           repo,
		publisher:      publisher,
		leaderElection: leaderElection,
		config:         config,
		stopCh:         make(chan struct{}),
	}
}

// Start begins the processing loop
func (p *Processor) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return nil
	}

	// Start the leader election
	if err := p.leaderElection.Start(ctx); err != nil {
		return fmt.Errorf("failed to start leader election: %w", err)
	}

	p.running = true
	p.wg.Add(1)

	go p.processLoop(ctx)

	return nil
}

func (p *Processor) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return nil
	}

	close(p.stopCh)
	p.wg.Wait()
	p.running = false

	if err := p.leaderElection.Stop(); err != nil {
		return fmt.Errorf("failed to stop leader election: %w", err)
	}

	return nil
}

// processLoop polls for and processes outbox messages
func (p *Processor) processLoop(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.PollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			if p.leaderElection.IsLeader() {
				if err := p.processBatch(ctx); err != nil {
					log.Printf("Error processing batch: %v", err)
				}
			}
		}
	}
}

// processBatch handles a group of outbox messages
func (p *Processor) processBatch(ctx context.Context) error {
	// Get pending messages
	messages, err := p.repo.GetPendingMessages(ctx, p.config.BatchSize)
	if err != nil {
		log.Printf("Failed to get pending messages: %v", err)
		return fmt.Errorf("failed to get pending messages: %w", err)
	}

	if len(messages) > 0 {
		log.Printf("Processing %d pending messages", len(messages))
	}

	// Process each message
	for _, msg := range messages {
		if err := p.processMessage(ctx, msg); err != nil {
			log.Printf("Error processing message %s: %v", msg.ID, err)
		}
	}

	return nil
}

// processMessage handles a single outbox message
func (p *Processor) processMessage(ctx context.Context, msg *model.OutboxMessage) error {
	// Mark the message as processing
	if err := p.repo.MarkMessageAsProcessing(ctx, msg.ID); err != nil {
		log.Printf("Failed to mark message %s as processing: %v", msg.ID, err)
		return fmt.Errorf("failed to mark message as processing: %w", err)
	}

	// Publish the message
	err := p.publisher.Publish(ctx, msg.Topic, msg.Payload)

	if err != nil {
		log.Printf("Failed to publish message %s: %v", msg.ID, err)
		// If publishing failed, mark the message as failed
		markErr := p.repo.MarkMessageAsFailed(ctx, msg.ID, err)
		if markErr != nil {
			log.Printf("Failed to mark message %s as failed: %v", msg.ID, markErr)
			return fmt.Errorf("failed to mark message as failed: %w", markErr)
		}

		// If we've exceeded the maximum number of retries, log an error
		if msg.RetryCount >= p.config.MaxRetries {
			log.Printf("Message %s exceeded maximum retries: %v", msg.ID, err)
		}

		return fmt.Errorf("failed to publish message: %w", err)
	}

	// If publishing succeeded, mark the message as completed
	if err := p.repo.MarkMessageAsCompleted(ctx, msg.ID); err != nil {
		log.Printf("Failed to mark message %s as completed: %v", msg.ID, err)
		return fmt.Errorf("failed to mark message as completed: %w", err)
	}

	return nil
}
