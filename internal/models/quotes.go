package models

import (
	"github.com/google/uuid"
	"time"
)

type Quotes struct {
	ID            uuid.UUID
	CurrencyPair  string
	Status        string
	Rate          float32
	ErrorMessage  string
	AttemptCount  int32
	NextAttemptAt time.Time
	LockedUntil   *time.Time
	LeaseToken    uuid.UUID
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type IdempotencyKey struct {
	Key          string
	RequestHash  string
	UpdateID     uuid.UUID
	ResponseCode int16
	ExpiresAt    time.Time
	CreatedAt    time.Time
}

type OutboxEvent struct {
	ID            int64
	QuoteID       uuid.UUID
	EventType     string
	Payload       []byte
	CreatedAt     time.Time
	PublishedAt   *time.Time
	AttemptCount  int32
	NextAttemptAt time.Time
}