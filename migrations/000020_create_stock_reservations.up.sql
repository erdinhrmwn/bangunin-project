CREATE TABLE stock_reservations (
    id          UUID PRIMARY KEY,
    variant_id  UUID NOT NULL REFERENCES product_variants(id),
    order_id    UUID NOT NULL REFERENCES orders(id),
    qty         INT NOT NULL CHECK (qty > 0),
    status      VARCHAR(20) NOT NULL DEFAULT 'active'
                CHECK (status IN ('active', 'converted', 'released')),
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_stock_reservations_variant_id ON stock_reservations(variant_id);
CREATE INDEX idx_stock_reservations_order_id ON stock_reservations(order_id);
CREATE INDEX idx_stock_reservations_status_expires_at ON stock_reservations(status, expires_at);
