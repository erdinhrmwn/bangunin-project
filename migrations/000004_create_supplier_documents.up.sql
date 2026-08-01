-- One row per (supplier_id, doc_type): re-uploading replaces the existing
-- row (status resets to pending) rather than keeping upload history.
CREATE TABLE supplier_documents (
    id              UUID PRIMARY KEY,
    supplier_id     UUID NOT NULL REFERENCES suppliers(id),
    doc_type        VARCHAR(10) NOT NULL CHECK (doc_type IN ('nib', 'ktp', 'npwp')),
    file_url        VARCHAR(500) NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'approved', 'rejected')),
    rejection_note  TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (supplier_id, doc_type)
);
