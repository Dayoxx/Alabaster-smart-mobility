package models

import (
	"time"

	"github.com/google/uuid"
)

type Terminal struct {
	ID           uuid.UUID  `json:"id"`
	DriverID     uuid.UUID  `json:"driver_id"`
	DeviceSerial string     `json:"device_serial"`
	VehiclePlate string     `json:"vehicle_plate"`
	TerminalType string     `json:"terminal_type"`
	VehicleType  string     `json:"vehicle_type"`
	LastLat      *float64   `json:"last_lat"`
	LastLng      *float64   `json:"last_lng"`
	LastSeenAt   *time.Time `json:"last_seen_at"`
	IsActive     bool       `json:"is_active"`
}
