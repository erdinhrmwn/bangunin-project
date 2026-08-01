CREATE TABLE carts (
    id          UUID PRIMARY KEY,
    user_id     UUID NOT NULL UNIQUE REFERENCES users(id),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE cart_items (
    id          UUID PRIMARY KEY,
    cart_id     UUID NOT NULL REFERENCES carts(id),
    variant_id  UUID NOT NULL REFERENCES product_variants(id),
    qty         INT NOT NULL CHECK (qty > 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (cart_id, variant_id)
);

CREATE INDEX idx_cart_items_cart_id ON cart_items(cart_id);
