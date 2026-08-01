CREATE TABLE banners (
    id         UUID PRIMARY KEY,
    image_url  TEXT NOT NULL,
    link       TEXT NOT NULL DEFAULT '',
    starts_at  TIMESTAMPTZ NOT NULL,
    ends_at    TIMESTAMPTZ NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    is_active  BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_banners_active_schedule ON banners(is_active, starts_at, ends_at);
