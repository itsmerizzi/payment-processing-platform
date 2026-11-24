package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"payment-processing-platform/internal/domain"
)

const maxIdempotencyKeyLen = 255

type CreatePaymentInput struct {
	IdempotencyKey string
	AmountCents    int64
	Currency       string
}

type CreatePayment struct {
	repo  PaymentRepository
	ids   IDGenerator
	clock Clock
}

func NewCreatePayment(repo PaymentRepository, ids IDGenerator, clock Clock) *CreatePayment {
	return &CreatePayment{repo: repo, ids: ids, clock: clock}
}

// Execute returns the payment and whether it was created by this call
// (false means an earlier request with the same key and body is being replayed).
func (uc *CreatePayment) Execute(ctx context.Context, in CreatePaymentInput) (*domain.Payment, bool, error) {
	if n := len(in.IdempotencyKey); n == 0 || n > maxIdempotencyKeyLen {
		return nil, false, ErrInvalidIdempotencyKey
	}
	id, err := uc.ids.NewID()
	if err != nil {
		return nil, false, fmt.Errorf("generate id: %w", err)
	}
	p, err := domain.NewPayment(id, in.AmountCents, in.Currency, uc.clock.Now())
	if err != nil {
		return nil, false, err
	}
	// Hash the normalized values so "brl" and "BRL" count as the same request.
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d|%s", p.AmountCents, p.Currency)))
	hash := hex.EncodeToString(sum[:])

	stored, storedHash, created, err := uc.repo.CreateIdempotent(ctx, p, in.IdempotencyKey, hash)
	if err != nil {
		return nil, false, err
	}
	if !created && storedHash != hash {
		return nil, false, ErrIdempotencyConflict
	}
	return stored, created, nil
}

type GetPayment struct {
	repo PaymentRepository
}

func NewGetPayment(repo PaymentRepository) *GetPayment {
	return &GetPayment{repo: repo}
}

func (uc *GetPayment) Execute(ctx context.Context, id string) (*domain.Payment, error) {
	return uc.repo.Get(ctx, id)
}
