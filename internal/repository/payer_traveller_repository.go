package repository

import (
	"context"
	"transportation/internal/models"

	"github.com/google/uuid"
)

type PayerTravellerRepository interface {
	Create(ctx context.Context, relationship models.PayerTraveller) (models.PayerTraveller, error)
	FindByPayerAndTraveller(ctx context.Context, payerID uuid.UUID, travellerID uuid.UUID) (*models.PayerTraveller, error)
	FindActiveByPayer(ctx context.Context, payerID uuid.UUID) ([]models.PayerTraveller, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
}
