package models

import (
	"time"

	"github.com/google/uuid"
)

type Payment struct {
	ID                uuid.UUID `json:"id"`
	TripID            uuid.UUID `json:"trip_id"`
	PassengerID       uuid.UUID `json:"passenger_id"`
	AccountID         uuid.UUID `json:"account_id"`
	Amount            int64     `json:"amount"`
	Status            string    `json:"status"`
	IdempotencyKey    string    `json:"idempotency_key"`
	Provider          string    `json:"provider"`
	ProviderReference *string   `json:"provider_reference,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
