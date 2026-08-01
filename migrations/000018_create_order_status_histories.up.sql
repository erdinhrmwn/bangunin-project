CREATE TABLE order_status_histories (
    id          UUID PRIMARY KEY,
    order_id    UUID NOT NULL REFERENCES orders(id),
    from_status VARCHAR(20) NOT NULL DEFAULT '',
    to_status   VARCHAR(20) NOT NULL,
    actor_type  VARCHAR(20) NOT NULL
                CHECK (actor_type IN ('user', 'supplier', 'admin', 'system')),
    actor_id    UUID,
    note        VARCHAR(500) NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_order_status_histories_order_id ON order_status_histories(order_id);
