CREATE TABLE suppliers (
    id                 UUID PRIMARY KEY,
    user_id            UUID NOT NULL UNIQUE REFERENCES users(id),
    store_name         VARCHAR(150) NOT NULL,
    slug               VARCHAR(180) NOT NULL UNIQUE,
    description        TEXT NOT NULL DEFAULT '',
    status             VARCHAR(20) NOT NULL DEFAULT 'draft'
                       CHECK (status IN ('draft', 'pending', 'approved', 'rejected', 'suspended')),
    origin_city_id     INT NOT NULL DEFAULT 0,
    pickup_address     TEXT NOT NULL DEFAULT '',
    own_fleet_enabled  BOOLEAN NOT NULL DEFAULT false,
    fleet_coverage_km  INT NOT NULL DEFAULT 0,
    fleet_flat_rate    NUMERIC(15, 2) NOT NULL DEFAULT 0,
    verified_at        TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_suppliers_status ON suppliers(status);
