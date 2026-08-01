CREATE TABLE reviews (
    id            UUID PRIMARY KEY,
    order_item_id UUID NOT NULL UNIQUE REFERENCES order_items(id),
    product_id    UUID NOT NULL REFERENCES products(id),
    user_id       UUID NOT NULL REFERENCES users(id),
    rating        SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment       TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_reviews_product_id ON reviews(product_id);
CREATE INDEX idx_reviews_user_id ON reviews(user_id);
