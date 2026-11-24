//go:build integration

package postgres

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"payment-processing-platform/internal/application"
	"payment-processing-platform/internal/domain"
)

func newRepo(t *testing.T) (*Repository, *pgxpool.Pool) {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return NewRepository(pool), pool
}

func newPayment(t *testing.T) *domain.Payment {
	t.Helper()
	id, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	p, err := domain.NewPayment(id.String(), 1500, "BRL", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestCreateAndGet(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	key := uuid.NewString()
	p := newPayment(t)

	stored, hash, created, err := repo.CreateIdempotent(ctx, p, key, "h1")
	if err != nil || !created || hash != "h1" || stored.ID != p.ID {
		t.Fatalf("stored=%+v hash=%q created=%v err=%v", stored, hash, created, err)
	}

	// Same key again with another payment: the original comes back.
	stored, hash, created, err = repo.CreateIdempotent(ctx, newPayment(t), key, "h2")
	if err != nil || created || hash != "h1" || stored.ID != p.ID {
		t.Fatalf("replay: stored=%+v hash=%q created=%v err=%v", stored, hash, created, err)
	}

	got, err := repo.Get(ctx, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.AmountCents != 1500 || got.Currency != "BRL" || got.Status != domain.StatusPending || !got.CreatedAt.Equal(p.CreatedAt.Truncate(time.Microsecond)) {
		t.Fatalf("got = %+v", got)
	}

	missing, _ := uuid.NewV7()
	if _, err := repo.Get(ctx, missing.String()); !errors.Is(err, application.ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestConcurrentSameKeyCreatesOne(t *testing.T) {
	repo, pool := newRepo(t)
	ctx := context.Background()
	key := uuid.NewString()

	const n = 20
	var wg sync.WaitGroup
	ids := make([]string, n)
	createdFlags := make([]bool, n)
	errs := make([]error, n)
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			stored, _, created, err := repo.CreateIdempotent(ctx, newPayment(t), key, "h")
			errs[i], createdFlags[i] = err, created
			if err == nil {
				ids[i] = stored.ID
			}
		}()
	}
	wg.Wait()

	createdCount := 0
	for i := range n {
		if errs[i] != nil {
			t.Fatalf("call %d: %v", i, errs[i])
		}
		if createdFlags[i] {
			createdCount++
		}
		if ids[i] != ids[0] {
			t.Fatalf("call %d returned id %s, want %s", i, ids[i], ids[0])
		}
	}
	if createdCount != 1 {
		t.Fatalf("created = %d, want 1", createdCount)
	}
	var rows int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM payments WHERE idempotency_key = $1`, key).Scan(&rows); err != nil || rows != 1 {
		t.Fatalf("rows = %d err = %v", rows, err)
	}
}
