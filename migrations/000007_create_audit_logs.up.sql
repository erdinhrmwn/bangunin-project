CREATE TABLE audit_logs (
    id           UUID PRIMARY KEY,
    actor_id     UUID NOT NULL REFERENCES users(id),
    action       VARCHAR(50) NOT NULL,
    entity_type  VARCHAR(50) NOT NULL,
    entity_id    UUID NOT NULL,
    metadata     JSONB NOT NULL DEFAULT '{}',
    ip_address   VARCHAR(45) NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_logs_entity ON audit_logs(entity_type, entity_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);
