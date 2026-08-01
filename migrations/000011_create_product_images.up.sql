CREATE TABLE product_images (
    id          UUID PRIMARY KEY,
    product_id  UUID NOT NULL REFERENCES products(id),
    url         TEXT NOT NULL,
    is_primary  BOOLEAN NOT NULL DEFAULT false,
    sort_order  INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_product_images_product_id ON product_images(product_id);
