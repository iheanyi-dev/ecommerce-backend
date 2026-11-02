-- +goose Up

CREATE TABLE payments (
    id UUID PRIMARY KEY,
    attempt_id UUID,
    reference VARCHAR(100) NOT NULL UNIQUE,
    payer_id UUID NOT NULL,
    purpose VARCHAR(30) NOT NULL
        CHECK (purpose IN ('order', 'subscription')),
    business_reference_type VARCHAR(50) NOT NULL,
    business_reference_id VARCHAR(100) NOT NULL,
    amount BIGINT NOT NULL CHECK (amount > 0),
    currency VARCHAR(3) NOT NULL CHECK (char_length(currency) = 3),
    status VARCHAR(30) NOT NULL
        CHECK (status IN ('pending', 'processing', 'succeeded', 'failed')),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT payments_business_reference_unique
        UNIQUE (business_reference_type, business_reference_id)
);

CREATE TABLE payment_attempts (
    id UUID PRIMARY KEY,
    payment_id UUID NOT NULL REFERENCES payments(id),
    provider VARCHAR(50) NOT NULL,
    provider_reference VARCHAR(150),
    customer_email VARCHAR(320),
    checkout_url TEXT,
    expires_at TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    status VARCHAR(40) NOT NULL
        CHECK (status IN (
            'pending',
            'processing',
            'succeeded',
            'failed',
            'requires_reconciliation'
        )),
    failure_reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT payment_attempts_payment_id_id_unique
        UNIQUE (payment_id, id)
);

CREATE UNIQUE INDEX payment_attempts_provider_reference_unique
    ON payment_attempts(provider, provider_reference)
    WHERE provider_reference IS NOT NULL;

CREATE INDEX payment_attempts_payment_id_idx
    ON payment_attempts(payment_id);

ALTER TABLE payments
    ADD CONSTRAINT payments_current_attempt_fk
    FOREIGN KEY (id, attempt_id)
    REFERENCES payment_attempts(payment_id, id)
    DEFERRABLE INITIALLY DEFERRED;

CREATE TABLE payouts (
    id UUID PRIMARY KEY,
    attempt_id UUID,
    reference VARCHAR(100) NOT NULL UNIQUE,
    business_reference_type VARCHAR(50),
    business_reference_id VARCHAR(100),
    store_id UUID NOT NULL,
    order_id UUID NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    beneficiary_snapshot_encrypted BYTEA,
    amount BIGINT NOT NULL CHECK (amount > 0),
    currency VARCHAR(3) NOT NULL CHECK (char_length(currency) = 3),
    status VARCHAR(30) NOT NULL
        CHECK (status IN ('pending', 'processing', 'succeeded', 'failed')),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT payouts_business_reference_pair
        CHECK (
            (business_reference_type IS NULL AND business_reference_id IS NULL)
            OR
            (business_reference_type IS NOT NULL AND business_reference_id IS NOT NULL)
        )
);

CREATE UNIQUE INDEX payouts_business_reference_unique
    ON payouts(business_reference_type, business_reference_id)
    WHERE business_reference_type IS NOT NULL
      AND business_reference_id IS NOT NULL;

CREATE TABLE payout_attempts (
    id UUID PRIMARY KEY,
    payout_id UUID NOT NULL REFERENCES payouts(id),
    provider VARCHAR(50) NOT NULL,
    provider_reference VARCHAR(150),
    status VARCHAR(40) NOT NULL
        CHECK (status IN (
            'pending',
            'processing',
            'succeeded',
            'failed',
            'requires_reconciliation'
        )),
    failure_reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT payout_attempts_payout_id_id_unique
        UNIQUE (payout_id, id)
);

CREATE UNIQUE INDEX payout_attempts_provider_reference_unique
    ON payout_attempts(provider, provider_reference)
    WHERE provider_reference IS NOT NULL;

CREATE INDEX payout_attempts_payout_id_idx
    ON payout_attempts(payout_id);

ALTER TABLE payouts
    ADD CONSTRAINT payouts_current_attempt_fk
    FOREIGN KEY (id, attempt_id)
    REFERENCES payout_attempts(payout_id, id)
    DEFERRABLE INITIALLY DEFERRED;

CREATE INDEX payments_payer_created_idx
    ON payments(payer_id, created_at DESC);

CREATE INDEX payouts_store_created_idx
    ON payouts(store_id, created_at DESC);


-- +goose Down

DROP TABLE IF EXISTS payout_attempts;
DROP TABLE IF EXISTS payouts;
DROP TABLE IF EXISTS payment_attempts;
DROP TABLE IF EXISTS payments;
