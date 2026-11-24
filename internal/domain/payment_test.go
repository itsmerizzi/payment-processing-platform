package domain

import (
	"errors"
	"testing"
	"time"
)

var t0 = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func TestNewPayment(t *testing.T) {
	tests := []struct {
		name         string
		id           string
		amount       int64
		currency     string
		wantCurrency string
		wantErr      error
	}{
		{"valid", "p1", 1000, "BRL", "BRL", nil},
		{"lowercase currency is normalized", "p1", 1, "usd", "USD", nil},
		{"empty id", "", 1000, "BRL", "", ErrInvalidID},
		{"zero amount", "p1", 0, "BRL", "", ErrInvalidAmount},
		{"negative amount", "p1", -5, "BRL", "", ErrInvalidAmount},
		{"currency too short", "p1", 1000, "BR", "", ErrInvalidCurrency},
		{"currency too long", "p1", 1000, "BRLL", "", ErrInvalidCurrency},
		{"currency with digits", "p1", 1000, "B2L", "", ErrInvalidCurrency},
		{"empty currency", "p1", 1000, "", "", ErrInvalidCurrency},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewPayment(tt.id, tt.amount, tt.currency, t0)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if err == nil && (p.Status != StatusPending || p.Currency != tt.wantCurrency || !p.CreatedAt.Equal(t0)) {
				t.Fatalf("unexpected payment: %+v", p)
			}
		})
	}
}

func TestTransitions(t *testing.T) {
	actions := map[string]func(*Payment, time.Time) error{
		"StartProcessing": (*Payment).StartProcessing,
		"Approve":         (*Payment).Approve,
		"Decline":         (*Payment).Decline,
		"Fail":            (*Payment).Fail,
	}
	// allowed[from][action] = resulting status; absent means ErrInvalidTransition.
	allowed := map[Status]map[string]Status{
		StatusPending:    {"StartProcessing": StatusProcessing},
		StatusProcessing: {"StartProcessing": StatusProcessing, "Approve": StatusApproved, "Decline": StatusDeclined, "Fail": StatusFailed},
		StatusApproved:   {},
		StatusDeclined:   {},
		StatusFailed:     {},
	}
	t1 := t0.Add(time.Minute)
	for from, m := range allowed {
		for name, act := range actions {
			t.Run(string(from)+"/"+name, func(t *testing.T) {
				p := &Payment{Status: from, UpdatedAt: t0}
				err := act(p, t1)
				if to, ok := m[name]; ok {
					if err != nil || p.Status != to || !p.UpdatedAt.Equal(t1) {
						t.Fatalf("err=%v status=%s updated=%v", err, p.Status, p.UpdatedAt)
					}
					return
				}
				if !errors.Is(err, ErrInvalidTransition) {
					t.Fatalf("err = %v, want ErrInvalidTransition", err)
				}
				if p.Status != from || !p.UpdatedAt.Equal(t0) {
					t.Fatalf("payment mutated on error: %+v", p)
				}
			})
		}
	}
}
