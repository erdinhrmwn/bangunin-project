CREATE TABLE shipments (
    id                UUID PRIMARY KEY,
    order_id          UUID NOT NULL UNIQUE REFERENCES orders(id),
    method            VARCHAR(20) NOT NULL
                      CHECK (method IN ('courier', 'supplier_fleet')),
    courier_code      VARCHAR(30) NOT NULL DEFAULT '',
    courier_service   VARCHAR(50) NOT NULL DEFAULT '',
    tracking_number   VARCHAR(100),
    cost              NUMERIC(15, 2) NOT NULL DEFAULT 0,
    status            VARCHAR(20) NOT NULL DEFAULT 'pending'
                      CHECK (status IN ('pending', 'picked_up', 'in_transit', 'delivered')),
    shipped_at        TIMESTAMPTZ,
    delivered_at      TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
