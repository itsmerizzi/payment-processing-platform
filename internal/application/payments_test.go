package application_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"payment-processing-platform/internal/application"
	"payment-processing-platform/internal/application/fakes"
	"payment-processing-platform/internal/domain"
)

func setup() (*application.CreatePayment, *application.GetPayment, *fakes.Repo) {
	repo := fakes.NewRepo()
	clock := fakes.FixedClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	return application.NewCreatePayment(repo, &fakes.SeqIDs{}, clock), application.NewGetPayment(repo), repo
}

func TestCreatePayment(t *testing.T) {
	ctx := context.Background()
	create, get, repo := setup()
	in := application.CreatePaymentInput{IdempotencyKey: "k1", AmountCents: 1000, Currency: "BRL"}

	p, created, err := create.Execute(ctx, in)
	if err != nil || !created || p.Status != domain.StatusPending {
		t.Fatalf("first call: p=%+v created=%v err=%v", p, created, err)
	}

	t.Run("same key and body returns the same payment", func(t *testing.T) {
		in2 := in
		in2.Currency = "brl" // normalized, so still the same request
		p2, created, err := create.Execute(ctx, in2)
		if err != nil || created || p2.ID != p.ID || repo.Len() != 1 {
			t.Fatalf("p2=%+v created=%v err=%v len=%d", p2, created, err, repo.Len())
		}
	})

	t.Run("same key with different body conflicts", func(t *testing.T) {
		in2 := in
		in2.AmountCents = 2000
		if _, _, err := create.Execute(ctx, in2); !errors.Is(err, application.ErrIdempotencyConflict) {
			t.Fatalf("err = %v", err)
		}
		if repo.Len() != 1 {
			t.Fatalf("len = %d", repo.Len())
		}
	})

	t.Run("new key creates a new payment", func(t *testing.T) {
		in2 := in
		in2.IdempotencyKey = "k2"
		p2, created, err := create.Execute(ctx, in2)
		if err != nil || !created || p2.ID == p.ID {
			t.Fatalf("p2=%+v created=%v err=%v", p2, created, err)
		}
	})

	t.Run("get", func(t *testing.T) {
		got, err := get.Execute(ctx, p.ID)
		if err != nil || got.ID != p.ID {
			t.Fatalf("got=%+v err=%v", got, err)
		}
		if _, err := get.Execute(ctx, "missing"); !errors.Is(err, application.ErrNotFound) {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestCreatePaymentValidation(t *testing.T) {
	create, _, repo := setup()
	tests := []struct {
		name string
		in   application.CreatePaymentInput
		want error
	}{
		{"empty key", application.CreatePaymentInput{"", 100, "BRL"}, application.ErrInvalidIdempotencyKey},
		{"long key", application.CreatePaymentInput{strings.Repeat("a", 256), 100, "BRL"}, application.ErrInvalidIdempotencyKey},
		{"zero amount", application.CreatePaymentInput{"k", 0, "BRL"}, domain.ErrInvalidAmount},
		{"bad currency", application.CreatePaymentInput{"k", 100, "XX"}, domain.ErrInvalidCurrency},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, err := create.Execute(context.Background(), tt.in); !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
		})
	}
	if repo.Len() != 0 {
		t.Fatalf("nothing should be stored, len = %d", repo.Len())
	}
}
