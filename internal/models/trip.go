package models

import (
	"github.com/google/uuid"
	"time"
)

type Trip struct {
	ID           uuid.UUID `json:"id"`
	PassengerID  uuid.UUID `json:"passenger_id"`
	DriverID     uuid.UUID `json:"driver_id"`
	TerminalID   uuid.UUID `json:"terminal_id"`
	FareAmount   float64   `json:"fare_amount"`
	DriverPayout float64   `json:"driver_payout"`
	PlatformFee  float64   `json:"platform_fee"`
	PickupLat    *float64  `json:"pickup_lat"`
	PickupLng    *float64  `json:"pickup_lng"`
	Status       string    `json:"status"`
	TappedAt     time.Time `json:"tapped_at"`
}
