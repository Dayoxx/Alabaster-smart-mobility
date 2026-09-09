package services

import (
	"context"

	"transportation/internal/models"
	"transportation/internal/repository"
)

type PaymentService struct {
	payments repository.PaymentRepository
}

func NewPaymentService(
	payments repository.PaymentRepository,
) *PaymentService {
	return &PaymentService{
		payments: payments,
	}
}

func (s *PaymentService) GetByIdempotencyKey(
	ctx context.Context,
	key string,
) (*models.Payment, error) {
	return s.payments.FindByIdempotencyKey(ctx, key)
}
