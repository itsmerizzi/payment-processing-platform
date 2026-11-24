// Package domain holds the payment entity and its state machine.
package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type Status string

const (
	StatusPending    Status = "PENDING"
	StatusProcessing Status = "PROCESSING"
	StatusApproved   Status = "APPROVED"
	StatusDeclined   Status = "DECLINED"
	StatusFailed     Status = "FAILED"
)

var (
	ErrInvalidID         = errors.New("id must not be empty")
	ErrInvalidAmount     = errors.New("amount must be greater than zero")
	ErrInvalidCurrency   = errors.New("currency must be a 3-letter ISO 4217 code")
	ErrInvalidTransition = errors.New("invalid status transition")
)

// Payment is the aggregate root. AmountCents is in minor units.
type Payment struct {
	ID          string
	AmountCents int64
	Currency    string
	Status      Status
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewPayment validates the input and returns a PENDING payment.
// The currency is normalized to upper case.
func NewPayment(id string, amountCents int64, currency string, now time.Time) (*Payment, error) {
	if id == "" {
		return nil, ErrInvalidID
	}
	if amountCents <= 0 {
		return nil, ErrInvalidAmount
	}
	currency = strings.ToUpper(currency)
	if len(currency) != 3 || strings.Trim(currency, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") != "" {
		return nil, ErrInvalidCurrency
	}
	return &Payment{
		ID: id, AmountCents: amountCents, Currency: currency,
		Status: StatusPending, CreatedAt: now, UpdatedAt: now,
	}, nil
}

// StartProcessing moves PENDING to PROCESSING. It is also allowed from
// PROCESSING so a redelivered message or a retry after a transient failure
// can reprocess the payment.
func (p *Payment) StartProcessing(now time.Time) error {
	return p.transition(now, StatusProcessing, StatusPending, StatusProcessing)
}

func (p *Payment) Approve(now time.Time) error {
	return p.transition(now, StatusApproved, StatusProcessing)
}

func (p *Payment) Decline(now time.Time) error {
	return p.transition(now, StatusDeclined, StatusProcessing)
}

// Fail marks a definitive technical failure (retries exhausted).
func (p *Payment) Fail(now time.Time) error {
	return p.transition(now, StatusFailed, StatusProcessing)
}

func (p *Payment) transition(now time.Time, to Status, from ...Status) error {
	for _, f := range from {
		if p.Status == f {
			p.Status = to
			p.UpdatedAt = now
			return nil
		}
	}
	return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, p.Status, to)
}
