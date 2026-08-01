CREATE TABLE stock_movements (
    id           UUID PRIMARY KEY,
    variant_id   UUID NOT NULL REFERENCES product_variants(id),
    type         VARCHAR(20) NOT NULL
                 CHECK (type IN ('in', 'out', 'adjustment', 'reserve_convert')),
    qty          INT NOT NULL,
    ref_type     VARCHAR(20) NOT NULL
                 CHECK (ref_type IN ('order', 'manual', 'import')),
    ref_id       UUID,
    stock_after  INT NOT NULL,
    note         TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_stock_movements_variant_id ON stock_movements(variant_id);
