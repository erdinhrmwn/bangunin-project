CREATE TABLE orders (
    id                  UUID PRIMARY KEY,
    checkout_group_id   UUID NOT NULL REFERENCES checkout_groups(id),
    user_id             UUID NOT NULL REFERENCES users(id),
    supplier_id         UUID NOT NULL REFERENCES suppliers(id),
    order_number        VARCHAR(30) NOT NULL UNIQUE,
    status              VARCHAR(20) NOT NULL DEFAULT 'pending_payment'
                        CHECK (status IN ('pending_payment', 'paid', 'processed', 'shipped', 'delivered', 'completed', 'expired', 'cancelled')),
    subtotal            NUMERIC(15, 2) NOT NULL DEFAULT 0,
    shipping_cost       NUMERIC(15, 2) NOT NULL DEFAULT 0,
    commission_amount   NUMERIC(15, 2) NOT NULL DEFAULT 0,
    total                NUMERIC(15, 2) NOT NULL DEFAULT 0,
    shipping_address    JSONB NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_orders_checkout_group_id ON orders(checkout_group_id);
CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_orders_supplier_id ON orders(supplier_id);

CREATE TABLE order_items (
    id            UUID PRIMARY KEY,
    order_id      UUID NOT NULL REFERENCES orders(id),
    variant_id    UUID NOT NULL REFERENCES product_variants(id),
    product_name  VARCHAR(200) NOT NULL,
    variant_name  VARCHAR(150) NOT NULL,
    unit          VARCHAR(30) NOT NULL,
    price         NUMERIC(15, 2) NOT NULL,
    qty           INT NOT NULL CHECK (qty > 0),
    weight_gram   INT NOT NULL DEFAULT 0
);

CREATE INDEX idx_order_items_order_id ON order_items(order_id);
