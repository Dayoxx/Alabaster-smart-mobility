package repository

import (
	"context"

	"transportation/internal/models"
)

type UserRepository interface {
	Create(ctx context.Context, u models.User) (models.User, error)
	FindByID(ctx context.Context, id string) (models.User, error)
}

type PassengerAccountRepository interface {
	Create(ctx context.Context, a models.PassengerAccount) (models.PassengerAccount, error)
	CountByPassenger(ctx context.Context, passengerID string) (int, error)
	FindActiveByPassenger(ctx context.Context, passengerID string) ([]models.PassengerAccount, error)
}

type TerminalRepository interface {
	Create(ctx context.Context, t models.Terminal) (models.Terminal, error)
	FindByID(ctx context.Context, id string) (*models.Terminal, error)
	FindByDeviceSerial(ctx context.Context, serial string) (*models.Terminal, error)
}

type TripRepository interface {
	Create(ctx context.Context, t models.Trip) (models.Trip, error)
	FindByPassengerOrDriver(ctx context.Context, passengerID, driverID string) ([]models.Trip, error)
}
