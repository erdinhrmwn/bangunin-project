CREATE TABLE user_addresses (
    id               UUID PRIMARY KEY,
    user_id          UUID NOT NULL REFERENCES users(id),
    label            VARCHAR(20) NOT NULL DEFAULT 'home'
                     CHECK (label IN ('home', 'project', 'warehouse')),
    recipient_name   VARCHAR(150) NOT NULL,
    recipient_phone  VARCHAR(20) NOT NULL,
    province_id      INT NOT NULL,
    city_id          INT NOT NULL,
    subdistrict      VARCHAR(100) NOT NULL DEFAULT '',
    postal_code      VARCHAR(10) NOT NULL DEFAULT '',
    address_detail   TEXT NOT NULL DEFAULT '',
    is_default       BOOLEAN NOT NULL DEFAULT false,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_user_addresses_user_id ON user_addresses(user_id);
