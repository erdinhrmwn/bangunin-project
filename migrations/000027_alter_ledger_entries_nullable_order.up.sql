-- debit_withdraw entries settle a withdraw_request, not an order.
ALTER TABLE ledger_entries ALTER COLUMN order_id DROP NOT NULL;
ALTER TABLE ledger_entries ADD COLUMN withdraw_request_id UUID REFERENCES withdraw_requests(id);

ALTER TABLE ledger_entries ADD CONSTRAINT ledger_entries_ref_check CHECK (
    (type IN ('credit_order', 'debit_commission') AND order_id IS NOT NULL AND withdraw_request_id IS NULL) OR
    (type = 'debit_withdraw' AND order_id IS NULL AND withdraw_request_id IS NOT NULL)
);
