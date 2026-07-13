CREATE TABLE IF NOT EXISTS payment_invoices (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES payment_orders(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title_name VARCHAR(200) NOT NULL,
    tax_id VARCHAR(32) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'REQUESTED',
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    issued_at TIMESTAMPTZ NULL,
    failed_at TIMESTAMPTZ NULL,
    failed_reason TEXT NULL,
    storage_provider VARCHAR(20) NOT NULL DEFAULT 'local',
    storage_key TEXT NULL,
    file_name VARCHAR(255) NULL,
    content_type VARCHAR(100) NULL,
    byte_size BIGINT NOT NULL DEFAULT 0,
    sha256 VARCHAR(64) NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT payment_invoices_order_id_key UNIQUE (order_id)
);

CREATE INDEX IF NOT EXISTS payment_invoices_user_id_idx
    ON payment_invoices (user_id);

CREATE INDEX IF NOT EXISTS payment_invoices_status_idx
    ON payment_invoices (status);

CREATE INDEX IF NOT EXISTS payment_invoices_requested_at_idx
    ON payment_invoices (requested_at);

CREATE INDEX IF NOT EXISTS payment_invoices_issued_at_idx
    ON payment_invoices (issued_at);

CREATE INDEX IF NOT EXISTS payment_invoices_created_at_idx
    ON payment_invoices (created_at);
