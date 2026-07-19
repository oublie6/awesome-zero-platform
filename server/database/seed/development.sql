INSERT INTO foundation_schema_meta (meta_key, meta_value)
VALUES ('seed_state', 'development')
ON CONFLICT (meta_key) DO UPDATE
SET meta_value = EXCLUDED.meta_value,
    updated_at = NOW();
