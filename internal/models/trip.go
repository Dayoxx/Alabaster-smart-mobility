package models

import (
	"github.com/google/uuid"
	"time"
)

type Trip struct {
	ID          uuid.UUID `json:"id"`
	PassengerID uuid.UUID `json:"passenger_id"`
	DriverID    uuid.UUID `json:"driver_id"`
	TerminalID  uuid.UUID `json:"terminal_id"`

	//All monetary values are in kobo.
	FareAmount           int64 `json:"fare_amount"`
	PassengerPlatformFee int64 `json:"passenger_platform_fee"`
	DriverPlatformFee    int64 `json:"driver_platform_fee"`
	PassengerTotal       int64 `json:"passenger_total"`
	DriverPayout         int64 `json:"driver_payout"`
	PlatformTotal        int64 `json:"platform_total"`

	PickupLat *float64 `json:"pickup_lat"`
	PickupLng *float64 `json:"pickup_lng"`

	Status   string    `json:"status"`
	TappedAt time.Time `json:"tapped_at"`
}
