CREATE TABLE notifications (
    id          UUID PRIMARY KEY,
    user_id     UUID NOT NULL REFERENCES users(id),
    type        VARCHAR(20) NOT NULL CHECK (type IN ('order_update', 'approval', 'payout', 'system')),
    title       VARCHAR(200) NOT NULL,
    body        TEXT NOT NULL DEFAULT '',
    data        JSONB NOT NULL DEFAULT '{}',
    read_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_notifications_user_id ON notifications(user_id);
