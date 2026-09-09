package repository

import (
	"context"

	"transportation/internal/models"

	"github.com/google/uuid"
)

type PaymentRepository interface {
	Create(ctx context.Context, payment *models.Payment) (models.Payment, error)
	FindByIdempotencyKey(ctx context.Context, key string) (*models.Payment, error)
	FindByID(ctx context.Context, id uuid.UUID) (*models.Payment, error)
}

//Any payment repository must know how to create a payment, find one by its idempotency key, and find one by its ID
