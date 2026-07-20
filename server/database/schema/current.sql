CREATE TABLE IF NOT EXISTS foundation_schema_meta (
    meta_key VARCHAR(191) NOT NULL,
    meta_value VARCHAR(255) NOT NULL,
    updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (meta_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS identity_accounts (
    account_id CHAR(36) NOT NULL,
    username VARCHAR(32) NULL,
    username_key VARCHAR(32) NULL,
    email VARCHAR(320) NULL,
    email_key VARCHAR(320) NULL,
    phone VARCHAR(16) NULL,
    phone_key VARCHAR(16) NULL,
    display_name VARCHAR(120) NOT NULL,
    status VARCHAR(16) NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (account_id),
    UNIQUE KEY uq_identity_accounts_username_key (username_key),
    UNIQUE KEY uq_identity_accounts_email_key (email_key),
    UNIQUE KEY uq_identity_accounts_phone_key (phone_key),
    KEY idx_identity_accounts_status (status),
    CONSTRAINT chk_identity_accounts_status CHECK (status IN ('active', 'disabled')),
    CONSTRAINT chk_identity_accounts_identity_present CHECK (
        username_key IS NOT NULL OR email_key IS NOT NULL OR phone_key IS NOT NULL
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS identity_password_credentials (
    account_id CHAR(36) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    password_changed_at DATETIME(6) NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (account_id),
    CONSTRAINT fk_identity_password_credentials_account
        FOREIGN KEY (account_id) REFERENCES identity_accounts (account_id)
        ON UPDATE RESTRICT
        ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO foundation_schema_meta (meta_key, meta_value)
VALUES ('schema_version', '0005')
ON DUPLICATE KEY UPDATE
    meta_value = VALUES(meta_value),
    updated_at = CURRENT_TIMESTAMP(6);
