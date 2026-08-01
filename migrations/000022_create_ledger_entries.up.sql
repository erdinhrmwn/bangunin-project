CREATE TABLE ledger_entries (
    id            UUID PRIMARY KEY,
    order_id      UUID NOT NULL REFERENCES orders(id),
    supplier_id   UUID NOT NULL REFERENCES suppliers(id),
    type          VARCHAR(20) NOT NULL
                  CHECK (type IN ('credit_order', 'debit_commission', 'debit_withdraw')),
    amount        NUMERIC(15, 2) NOT NULL,
    balance_after NUMERIC(15, 2) NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_ledger_entries_supplier_id ON ledger_entries(supplier_id);
CREATE INDEX idx_ledger_entries_order_id ON ledger_entries(order_id);

-- append-only: no role separation in this app (single DB user), so enforce
-- immutability with a trigger instead of REVOKE.
CREATE FUNCTION ledger_entries_no_mutate() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'ledger_entries is append-only';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER ledger_entries_no_mutate_trigger
    BEFORE UPDATE OR DELETE ON ledger_entries
    FOR EACH ROW EXECUTE FUNCTION ledger_entries_no_mutate();
