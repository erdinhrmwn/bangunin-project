CREATE TABLE withdraw_requests (
    id               UUID PRIMARY KEY,
    supplier_id      UUID NOT NULL REFERENCES suppliers(id),
    bank_account_id  UUID NOT NULL REFERENCES supplier_bank_accounts(id),
    amount           NUMERIC(15, 2) NOT NULL CHECK (amount > 0),
    status           VARCHAR(20) NOT NULL DEFAULT 'pending'
                     CHECK (status IN ('pending', 'approved', 'rejected', 'disbursed', 'failed')),
    disbursement_id  VARCHAR(100) NOT NULL DEFAULT '',
    requested_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    processed_at     TIMESTAMPTZ
);

CREATE INDEX idx_withdraw_requests_supplier_id ON withdraw_requests(supplier_id);
CREATE INDEX idx_withdraw_requests_status ON withdraw_requests(status);
