package models

import "time"

// User mirrors the `users` table.
type User struct {
	ID        string    `json:"id"`
	Phone     string    `json:"phone"`
	FullName  string    `json:"full_name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// PassengerAccount mirrors the `passenger_accounts` table.
type PassengerAccount struct {
	ID            string `json:"id"`
	PassengerID   string `json:"passenger_id"`
	AccountNumber string `json:"account_number"`
	BankCode      string `json:"bank_code"`
	BankName      string `json:"bank_name"`
	PriorityOrder int    `json:"priority_order"`
	IsActive      bool   `json:"is_active"`
}

// Terminal mirrors the `terminals` table.
type Terminal struct {
	ID           string    `json:"id"`
	DriverID     string    `json:"driver_id"`
	DeviceSerial string    `json:"device_serial"`
	VehiclePlate string    `json:"vehicle_plate"`
	TerminalType string    `json:"terminal_type"` // "keke" or "bus" only, for now
	LastLat      float64   `json:"last_lat"`
	LastLng      float64   `json:"last_lng"`
	LastSeenAt   time.Time `json:"last_seen_at"`
	IsActive     bool      `json:"is_active"`
}

// Trip mirrors the `trips` table.
type Trip struct {
	ID           string    `json:"id"`
	PassengerID  string    `json:"passenger_id"`
	DriverID     string    `json:"driver_id"`
	TerminalID   string    `json:"terminal_id"`
	FareAmount   float64   `json:"fare_amount"`
	DriverPayout float64   `json:"driver_payout"`
	PlatformFee  float64   `json:"platform_fee"`
	PickupLat    float64   `json:"pickup_lat"`
	PickupLng    float64   `json:"pickup_lng"`
	Status       string    `json:"status"`
	TappedAt     time.Time `json:"tapped_at"`
}
