CREATE TABLE products (
    id             UUID PRIMARY KEY,
    supplier_id    UUID NOT NULL REFERENCES suppliers(id),
    category_id    INT NOT NULL REFERENCES categories(id),
    name           VARCHAR(200) NOT NULL,
    slug           VARCHAR(220) NOT NULL UNIQUE,
    description    TEXT NOT NULL DEFAULT '',
    specs          JSONB NOT NULL DEFAULT '{}',
    status         VARCHAR(20) NOT NULL DEFAULT 'draft'
                   CHECK (status IN ('draft', 'active', 'inactive', 'banned')),
    rating_avg     NUMERIC(3, 2) NOT NULL DEFAULT 0,
    rating_count   INT NOT NULL DEFAULT 0,
    search_vector  TSVECTOR,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_products_supplier_id ON products(supplier_id);
CREATE INDEX idx_products_category_id ON products(category_id);
CREATE INDEX idx_products_status ON products(status);
CREATE INDEX idx_products_search_vector ON products USING GIN (search_vector);

-- 'indonesian' text search config ships with stock Postgres 16 (pg_catalog),
-- no extension needed — verified against postgres:16-alpine.
CREATE FUNCTION products_search_vector_update() RETURNS trigger AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('indonesian', coalesce(NEW.name, '')), 'A') ||
        setweight(to_tsvector('indonesian', coalesce(NEW.description, '')), 'B') ||
        setweight(to_tsvector('indonesian', coalesce(NEW.specs->>'brand', '')), 'C');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER products_search_vector_trigger
    BEFORE INSERT OR UPDATE ON products
    FOR EACH ROW EXECUTE FUNCTION products_search_vector_update();
