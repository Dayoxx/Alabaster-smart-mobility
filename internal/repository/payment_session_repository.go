package repository

import (
	"context"
	"transportation/internal/models"

	"github.com/google/uuid"
)

type PaymentSessionRepository interface {
	Create(
		ctx context.Context,
		session models.PaymentSession,
	) (models.PaymentSession, error)

	FindByIdempotencyKey(
		ctx context.Context,
		key string,
	) (*models.PaymentSession, error)

	FindByID(
		ctx context.Context,
		id uuid.UUID,
	) (*models.PaymentSession, error)
}
