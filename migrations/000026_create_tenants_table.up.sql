CREATE TABLE tenants (
    id              BIGSERIAL PRIMARY KEY,
    name            VARCHAR(255) NOT NULL,
    slug            VARCHAR(64)  NOT NULL,
    is_active       BOOLEAN      NOT NULL DEFAULT TRUE,
    ghost_order_timeout_minutes INT NOT NULL DEFAULT 30,
    -- Suscripción actual (snapshot sencillo)
    subscription_status VARCHAR(16) NOT NULL DEFAULT 'active',
    current_period_end  TIMESTAMPTZ,
    plan_code           VARCHAR(32) DEFAULT 'basic',
    -- Branding
    logo_url        TEXT,
    logo_width      INT,
    logo_height     INT,
    primary_color   CHAR(7),
    secondary_color CHAR(7),
    accent_color    CHAR(7),
    created_on      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_on      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX ux_tenants_slug ON tenants (slug);

INSERT INTO tenants (
    name,
    slug,
    is_active,
    ghost_order_timeout_minutes,
    subscription_status,
    plan_code
) VALUES (
    'Default Tenant',
    'default',
    TRUE,
    30,
    'active',
    'basic'
);

