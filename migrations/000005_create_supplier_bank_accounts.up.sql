CREATE TABLE supplier_bank_accounts (
    id              UUID PRIMARY KEY,
    supplier_id     UUID NOT NULL REFERENCES suppliers(id),
    bank_code       VARCHAR(20) NOT NULL,
    account_number  VARCHAR(20) NOT NULL,
    account_name    VARCHAR(150) NOT NULL,
    is_default      BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_supplier_bank_accounts_supplier_id ON supplier_bank_accounts(supplier_id);
