CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX idx_products_lower_name_trgm
    ON products USING gin (lower(name) gin_trgm_ops);
