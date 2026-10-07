package models

import (
	"time"

	"github.com/google/uuid"
)

type PaymentSession struct {
	ID             uuid.UUID `json:"id"`
	PayerID        uuid.UUID `json:"payer_id"`
	TotalAmount    int64     `json:"total_amount"`
	Status         string    `json:"status"`
	IdempotencyKey string    `json:"idempotency_key"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
