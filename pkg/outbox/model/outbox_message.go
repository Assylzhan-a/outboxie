package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type OutboxMessageStatus string

const (
	StatusPending    OutboxMessageStatus = "pending"    // Message waiting to be processed
	StatusProcessing OutboxMessageStatus = "processing" // Message being processed
	StatusCompleted  OutboxMessageStatus = "completed"  // Message successfully processed
	StatusFailed     OutboxMessageStatus = "failed"     // Message processing failed
)

type OutboxMessage struct {
	ID             uuid.UUID           `json:"id"`
	Topic          string              `json:"topic"`
	Payload        json.RawMessage     `json:"payload"`
	CreatedAt      time.Time           `json:"created_at"`
	ProcessedAt    *time.Time          `json:"processed_at"`
	Status         OutboxMessageStatus `json:"status"`
	RetryCount     int                 `json:"retry_count"`
	Error          *string             `json:"error"`
	SequenceNumber int64               `json:"sequence_number"`
}

func NewOutboxMessage(topic string, payload interface{}) (*OutboxMessage, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &OutboxMessage{
		ID:         uuid.New(),
		Topic:      topic,
		Payload:    payloadBytes,
		CreatedAt:  time.Now().UTC(),
		Status:     StatusPending,
		RetryCount: 0,
	}, nil
}
