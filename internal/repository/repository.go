package repository

import (
	"context"
	"transportation/internal/models"

	"github.com/google/uuid"
)

type UserRepository interface {
	Create(ctx context.Context, u models.User) (models.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (models.User, error)
}

type PassengerAccountRepository interface {
	Create(ctx context.Context, a models.PassengerAccount) (models.PassengerAccount, error)
	CountByPassenger(ctx context.Context, passengerID uuid.UUID) (int, error)
	FindActiveByPassenger(ctx context.Context, passengerID uuid.UUID) ([]models.PassengerAccount, error)
}

type TerminalRepository interface {
	Create(ctx context.Context, t models.Terminal) (models.Terminal, error)
	FindByID(ctx context.Context, id uuid.UUID) (*models.Terminal, error)
	FindByDeviceSerial(ctx context.Context, serial string) (*models.Terminal, error)
}

type TripRepository interface {
	Create(ctx context.Context, t models.Trip) (models.Trip, error)
	FindByPassenger(ctx context.Context, passengerID uuid.UUID) ([]models.Trip, error)
	FindByDriver(ctx context.Context, driverID uuid.UUID) ([]models.Trip, error)
}
