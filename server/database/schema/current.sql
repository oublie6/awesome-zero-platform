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

CREATE TABLE IF NOT EXISTS authorization_casbin_rules (
    rule_id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    rule_hash CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    ptype VARCHAR(16) NOT NULL,
    v0 VARCHAR(191) NOT NULL DEFAULT '',
    v1 VARCHAR(191) NOT NULL DEFAULT '',
    v2 VARCHAR(191) NOT NULL DEFAULT '',
    v3 VARCHAR(191) NOT NULL DEFAULT '',
    v4 VARCHAR(191) NOT NULL DEFAULT '',
    v5 VARCHAR(191) NOT NULL DEFAULT '',
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (rule_id),
    UNIQUE KEY uq_authorization_casbin_rule_hash (rule_hash),
    KEY idx_authorization_casbin_subject (ptype, v0),
    KEY idx_authorization_casbin_object_action (ptype, v1, v2)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS authorization_roles (
    role_code VARCHAR(96) NOT NULL,
    display_name VARCHAR(120) NOT NULL,
    description VARCHAR(500) NOT NULL DEFAULT '',
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (role_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS authorization_resources (
    resource_code VARCHAR(120) NOT NULL,
    display_name VARCHAR(120) NOT NULL,
    description VARCHAR(500) NOT NULL DEFAULT '',
    resource_pattern VARCHAR(191) NOT NULL,
    actions_json JSON NOT NULL,
    module_name VARCHAR(96) NOT NULL,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (resource_code),
    KEY idx_authorization_resources_module (module_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS platform_audit_events (
    event_id CHAR(36) NOT NULL,
    action_name VARCHAR(96) NOT NULL,
    actor_account_id CHAR(36) NULL,
    resource_type VARCHAR(96) NOT NULL,
    resource_id VARCHAR(191) NULL,
    request_id VARCHAR(96) NULL,
    client_ip VARCHAR(64) NULL,
    user_agent VARCHAR(500) NULL,
    outcome VARCHAR(32) NOT NULL,
    details_json JSON NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (event_id),
    KEY idx_platform_audit_created_at (created_at),
    KEY idx_platform_audit_actor_created (actor_account_id, created_at),
    KEY idx_platform_audit_type_created (action_name, created_at),
    CONSTRAINT fk_platform_audit_actor
        FOREIGN KEY (actor_account_id) REFERENCES identity_accounts (account_id)
        ON UPDATE RESTRICT
        ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO authorization_roles (role_code, display_name, description, is_system)
VALUES
    ('platform_super_admin', '超级管理员', '拥有平台管理端全部权限。', TRUE),
    ('platform_security_admin', '安全管理员', '管理账号、角色、权限和安全审计。', TRUE),
    ('platform_operator', '平台运维员', '查看运行状态并执行受控运维操作。', TRUE),
    ('platform_viewer', '只读观察员', '只读查看平台状态和管理数据。', TRUE)
ON DUPLICATE KEY UPDATE
    display_name = VALUES(display_name),
    description = VALUES(description),
    is_system = VALUES(is_system);

INSERT INTO authorization_resources (
    resource_code, display_name, description, resource_pattern, actions_json, module_name, is_system
)
VALUES
    ('admin.dashboard', '管理首页', '平台运行概览和当前管理员信息。', '/admin/system/*', JSON_ARRAY('GET'), 'dashboard', TRUE),
    ('admin.account', '账号管理', '账号资料、状态、密码、角色和会话。', '/admin/accounts/*', JSON_ARRAY('GET', 'POST', 'PATCH', 'DELETE'), 'identity', TRUE),
    ('admin.role', '角色管理', '角色元数据、成员和标准权限。', '/admin/roles/*', JSON_ARRAY('GET', 'POST', 'PUT', 'PATCH', 'DELETE'), 'authorization', TRUE),
    ('admin.authorization', '权限配置', '资源目录和标准化权限配置。', '/admin/authorization/resources/*', JSON_ARRAY('GET', 'POST', 'PUT', 'PATCH', 'DELETE'), 'authorization', TRUE),
    ('admin.authorization.engine', '权限引擎', '权限插件模型、原始策略和执行解释。', '/admin/authorization/engine/*', JSON_ARRAY('GET', 'POST', 'PUT'), 'authorization', TRUE),
    ('admin.audit', '审计日志', '平台管理操作的安全审计事件。', '/admin/audit/*', JSON_ARRAY('GET'), 'security', TRUE)
ON DUPLICATE KEY UPDATE
    display_name = VALUES(display_name),
    description = VALUES(description),
    resource_pattern = VALUES(resource_pattern),
    actions_json = VALUES(actions_json),
    module_name = VALUES(module_name),
    is_system = VALUES(is_system);

INSERT IGNORE INTO authorization_casbin_rules (
    rule_hash, ptype, v0, v1, v2, v3, v4, v5
) VALUES (
    '8522ffb77fa9bd9d1a92da0854ec05f6dea81a526737cb1dfe29e3a898d5962f',
    'p', 'platform_super_admin', '/*', '.*', '', '', ''
);

INSERT INTO foundation_schema_meta (meta_key, meta_value)
VALUES ('schema_version', '0007')
ON DUPLICATE KEY UPDATE
    meta_value = VALUES(meta_value),
    updated_at = CURRENT_TIMESTAMP(6);
