// Package fakes has in-memory test doubles for the application ports.
package fakes

import (
	"context"
	"fmt"
	"sync"
	"time"

	"payment-processing-platform/internal/application"
	"payment-processing-platform/internal/domain"
)

type entry struct {
	p    *domain.Payment
	hash string
}

type Repo struct {
	mu    sync.Mutex
	byKey map[string]entry
	byID  map[string]*domain.Payment
}

func NewRepo() *Repo {
	return &Repo{byKey: map[string]entry{}, byID: map[string]*domain.Payment{}}
}

func (r *Repo) CreateIdempotent(_ context.Context, p *domain.Payment, key, hash string) (*domain.Payment, string, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if e, ok := r.byKey[key]; ok {
		return e.p, e.hash, false, nil
	}
	r.byKey[key] = entry{p, hash}
	r.byID[p.ID] = p
	return p, hash, true, nil
}

func (r *Repo) Get(_ context.Context, id string) (*domain.Payment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.byID[id]; ok {
		return p, nil
	}
	return nil, application.ErrNotFound
}

// Len is the number of stored payments.
func (r *Repo) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.byID)
}

// SeqIDs returns "id-1", "id-2", ...
type SeqIDs struct {
	mu sync.Mutex
	n  int
}

func (s *SeqIDs) NewID() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.n++
	return fmt.Sprintf("id-%d", s.n), nil
}

type FixedClock time.Time

func (c FixedClock) Now() time.Time { return time.Time(c) }
