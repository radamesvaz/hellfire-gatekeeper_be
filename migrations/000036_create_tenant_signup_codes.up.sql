CREATE TABLE tenant_signup_codes (
    id BIGSERIAL PRIMARY KEY,
    code_hash VARCHAR(128) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ NULL,
    revoked_at TIMESTAMPTZ NULL,
    created_by_user_id BIGINT NULL,
    used_by_tenant_id BIGINT NULL,
    used_by_user_id BIGINT NULL,
    used_email VARCHAR(320) NULL,
    notes TEXT NULL,
    created_on TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_on TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ux_tenant_signup_codes_code_hash UNIQUE (code_hash),
    CONSTRAINT fk_tenant_signup_codes_created_by_user
        FOREIGN KEY (created_by_user_id) REFERENCES users(id_user) ON DELETE SET NULL,
    CONSTRAINT fk_tenant_signup_codes_used_by_user
        FOREIGN KEY (used_by_user_id) REFERENCES users(id_user) ON DELETE SET NULL,
    CONSTRAINT fk_tenant_signup_codes_used_by_tenant
        FOREIGN KEY (used_by_tenant_id) REFERENCES tenants(id) ON DELETE SET NULL
);

CREATE INDEX idx_tenant_signup_codes_status
    ON tenant_signup_codes (used_at, revoked_at, expires_at);

CREATE INDEX idx_tenant_signup_codes_used_by_user
    ON tenant_signup_codes (used_by_user_id);

CREATE INDEX idx_tenant_signup_codes_used_by_tenant
    ON tenant_signup_codes (used_by_tenant_id);
