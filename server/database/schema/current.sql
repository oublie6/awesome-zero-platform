CREATE TABLE IF NOT EXISTS foundation_schema_meta (
    meta_key TEXT PRIMARY KEY,
    meta_value TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO foundation_schema_meta (meta_key, meta_value)
VALUES ('schema_version', '0003')
ON CONFLICT (meta_key) DO UPDATE
SET meta_value = EXCLUDED.meta_value,
    updated_at = NOW();
