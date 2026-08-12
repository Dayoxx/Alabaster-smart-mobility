package models

import "github.com/google/uuid"

type PassengerAccount struct {
	ID            uuid.UUID `json:"id"`
	PassengerID   uuid.UUID `json:"passenger_id"`
	AccountNumber string    `json:"-"`
	BankCode      string    `json:"bank_code"`
	BankName      string    `json:"bank_name"`
	PriorityOrder int       `json:"priority_order"`
	IsActive      bool      `json:"is_active"`
}
