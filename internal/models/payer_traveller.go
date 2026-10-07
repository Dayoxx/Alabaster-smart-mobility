package models

import (
	"time"

	"github.com/google/uuid"
)

type PayerTraveller struct {
	ID          uuid.UUID `json:"id"`
	PayerID     uuid.UUID `json:"payer_id"`
	TravellerID uuid.UUID `json:"traveller_id"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
