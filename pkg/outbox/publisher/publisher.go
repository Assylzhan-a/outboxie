package publisher

import (
	"context"
	"fmt"
	"sync"

	"github.com/nats-io/nats.go"
)

type Publisher interface {
	Publish(ctx context.Context, topic string, payload []byte) error

	Close() error
}

type NatsPublisher struct {
	conn *nats.Conn
	mu   sync.Mutex
}

func NewNatsPublisher(natsURL string) (*NatsPublisher, error) {
	conn, err := nats.Connect(natsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	return &NatsPublisher{
		conn: conn,
	}, nil
}

func (p *NatsPublisher) Publish(ctx context.Context, topic string, payload []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conn == nil || p.conn.IsClosed() {
		return fmt.Errorf("NATS connection is closed")
	}

	err := p.conn.Publish(topic, payload)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

func (p *NatsPublisher) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conn != nil && !p.conn.IsClosed() {
		p.conn.Close()
	}

	return nil
}
