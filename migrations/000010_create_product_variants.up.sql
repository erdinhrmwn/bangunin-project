CREATE TABLE product_variants (
    id             UUID PRIMARY KEY,
    product_id     UUID NOT NULL REFERENCES products(id),
    supplier_id    UUID NOT NULL REFERENCES suppliers(id),
    sku            VARCHAR(100) NOT NULL,
    name           VARCHAR(150) NOT NULL,
    unit           VARCHAR(20) NOT NULL
                   CHECK (unit IN ('sak', 'batang', 'm3', 'dus', 'pcs', 'lembar', 'kg', 'roll', 'set')),
    price          NUMERIC(15, 2) NOT NULL DEFAULT 0,
    weight_gram    INT NOT NULL CHECK (weight_gram > 0),
    length_cm      INT NOT NULL DEFAULT 0,
    width_cm       INT NOT NULL DEFAULT 0,
    height_cm      INT NOT NULL DEFAULT 0,
    min_order_qty  INT NOT NULL DEFAULT 1 CHECK (min_order_qty >= 1),
    stock          INT NOT NULL DEFAULT 0 CHECK (stock >= 0),
    is_active      BOOLEAN NOT NULL DEFAULT true,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (supplier_id, sku)
);

CREATE INDEX idx_product_variants_product_id ON product_variants(product_id);
