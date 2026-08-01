CREATE TABLE checkout_groups (
    id            UUID PRIMARY KEY,
    user_id       UUID NOT NULL REFERENCES users(id),
    group_number  VARCHAR(30) NOT NULL UNIQUE,
    grand_total   NUMERIC(15, 2) NOT NULL DEFAULT 0,
    status        VARCHAR(20) NOT NULL DEFAULT 'pending_payment'
                  CHECK (status IN ('pending_payment', 'paid', 'payment_failed', 'expired', 'cancelled')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_checkout_groups_user_id ON checkout_groups(user_id);
