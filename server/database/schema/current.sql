CREATE TABLE IF NOT EXISTS foundation_schema_meta (
    meta_key VARCHAR(191) NOT NULL,
    meta_value VARCHAR(255) NOT NULL,
    updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (meta_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO foundation_schema_meta (meta_key, meta_value)
VALUES ('schema_version', '0004')
ON DUPLICATE KEY UPDATE
    meta_value = VALUES(meta_value),
    updated_at = CURRENT_TIMESTAMP(6);
