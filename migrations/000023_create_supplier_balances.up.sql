CREATE TABLE supplier_balances (
    supplier_id UUID PRIMARY KEY REFERENCES suppliers(id),
    balance     NUMERIC(15, 2) NOT NULL DEFAULT 0,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
