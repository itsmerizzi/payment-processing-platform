// Package postgres implements application.PaymentRepository with pgx.
package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"payment-processing-platform/internal/application"
	"payment-processing-platform/internal/domain"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const selectCols = `id::text, amount_cents, currency, status, created_at, updated_at`

func scan(row pgx.Row, extra ...any) (*domain.Payment, error) {
	var p domain.Payment
	var status string
	dest := append([]any{&p.ID, &p.AmountCents, &p.Currency, &status, &p.CreatedAt, &p.UpdatedAt}, extra...)
	if err := row.Scan(dest...); err != nil {
		return nil, err
	}
	p.Status = domain.Status(status)
	return &p, nil
}

// CreateIdempotent relies on the UNIQUE constraint on idempotency_key: concurrent
// inserts with the same key serialize in the database and only one inserts a row.
func (r *Repository) CreateIdempotent(ctx context.Context, p *domain.Payment, key, requestHash string) (*domain.Payment, string, bool, error) {
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO payments (id, idempotency_key, request_hash, amount_cents, currency, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (idempotency_key) DO NOTHING`,
		p.ID, key, requestHash, p.AmountCents, p.Currency, string(p.Status), p.CreatedAt, p.UpdatedAt)
	if err != nil {
		return nil, "", false, err
	}
	if tag.RowsAffected() == 1 {
		return p, requestHash, true, nil
	}
	var storedHash string
	stored, err := scan(r.pool.QueryRow(ctx,
		`SELECT `+selectCols+`, request_hash FROM payments WHERE idempotency_key = $1`, key), &storedHash)
	if err != nil {
		return nil, "", false, err
	}
	return stored, storedHash, false, nil
}

func (r *Repository) Get(ctx context.Context, id string) (*domain.Payment, error) {
	p, err := scan(r.pool.QueryRow(ctx, `SELECT `+selectCols+` FROM payments WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, application.ErrNotFound
	}
	return p, err
}
