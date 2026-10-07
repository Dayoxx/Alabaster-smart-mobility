package models

import (
	"time"

	"github.com/google/uuid"
)

type PaymentAllocation struct {
	ID        uuid.UUID `json:"id"`
	PaymentID uuid.UUID `json:"payment_id"`
	TripID    uuid.UUID `json:"trip_id"`
	Amount    int64     `json:"amount"`
	CreatedAt time.Time `json:"created_at"`
}
