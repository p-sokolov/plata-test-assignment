package models

import (
	"time"

	"github.com/google/uuid"
)

type QuoteUpdate struct {
	ID            uuid.UUID
	CurrencyPair  string
	Status        string
	Rate          *float64
	ErrorMessage  *string
	AttemptCount  int32
	NextAttemptAt time.Time
	LockedUntil   *time.Time
	LeaseToken    *uuid.UUID
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type RefreshInput struct {
	CurrencyPair   string
	IdempotencyKey string
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
