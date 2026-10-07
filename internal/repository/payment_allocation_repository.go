package repository

import (
	"context"
	"transportation/internal/models"

	"github.com/google/uuid"
)

type PaymentAllocationRepository interface {
	Create(
		ctx context.Context,
		allocation models.PaymentAllocation,
	) (models.PaymentAllocation, error)

	FindByTripID(
		ctx context.Context,
		tripID uuid.UUID,
	) (*models.PaymentAllocation, error)

	FindByPaymentID(
		ctx context.Context,
		paymentID uuid.UUID,
	) ([]models.PaymentAllocation, error)
}
