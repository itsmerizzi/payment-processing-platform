CREATE TABLE payments (
    id              uuid PRIMARY KEY,
    idempotency_key text        NOT NULL,
    request_hash    text        NOT NULL,
    amount_cents    bigint      NOT NULL CHECK (amount_cents > 0),
    currency        text        NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    status          text        NOT NULL CHECK (status IN ('PENDING', 'PROCESSING', 'APPROVED', 'DECLINED', 'FAILED')),
    created_at      timestamptz NOT NULL,
    updated_at      timestamptz NOT NULL,
    -- Guarantees at most one payment per key, even under concurrent requests.
    CONSTRAINT payments_idempotency_key_key UNIQUE (idempotency_key)
);
