ALTER TABLE ledger_entries DROP CONSTRAINT ledger_entries_ref_check;
ALTER TABLE ledger_entries DROP COLUMN withdraw_request_id;
ALTER TABLE ledger_entries ALTER COLUMN order_id SET NOT NULL;
