INSERT INTO foundation_schema_meta (meta_key, meta_value)
VALUES ('seed_state', 'development')
ON DUPLICATE KEY UPDATE
    meta_value = VALUES(meta_value),
    updated_at = CURRENT_TIMESTAMP(6);
