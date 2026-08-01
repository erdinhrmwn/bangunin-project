CREATE TABLE payments (
    id                 UUID PRIMARY KEY,
    checkout_group_id  UUID NOT NULL UNIQUE REFERENCES checkout_groups(id),
    xendit_invoice_id  VARCHAR(100) NOT NULL UNIQUE,
    method             VARCHAR(20) NOT NULL DEFAULT ''
                       CHECK (method IN ('', 'va', 'ewallet', 'qris')),
    amount             NUMERIC(15, 2) NOT NULL,
    status             VARCHAR(20) NOT NULL DEFAULT 'pending'
                       CHECK (status IN ('pending', 'paid', 'expired', 'failed')),
    invoice_url        VARCHAR(500) NOT NULL DEFAULT '',
    paid_at            TIMESTAMPTZ,
    expires_at         TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
