// Package application holds the use cases and the ports they depend on.
package application

import (
	"context"
	"errors"
	"time"

	"payment-processing-platform/internal/domain"
)

var (
	ErrNotFound              = errors.New("payment not found")
	ErrIdempotencyConflict   = errors.New("idempotency key already used with a different request")
	ErrInvalidIdempotencyKey = errors.New("Idempotency-Key must have between 1 and 255 characters")
)

type PaymentRepository interface {
	// CreateIdempotent stores p under key. If the key already exists nothing is
	// written: the stored payment and its request hash come back with created=false.
	// It must be safe under concurrent calls with the same key.
	CreateIdempotent(ctx context.Context, p *domain.Payment, key, requestHash string) (stored *domain.Payment, storedHash string, created bool, err error)
	// Get returns ErrNotFound when the payment does not exist.
	Get(ctx context.Context, id string) (*domain.Payment, error)
}

type IDGenerator interface {
	NewID() (string, error)
}

type Clock interface {
	Now() time.Time
}
