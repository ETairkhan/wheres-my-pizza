CREATE TABLE IF NOT EXISTS order_status_log (
    log_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    order_id UUID REFERENCES orders(order_id) ON DELETE CASCADE,
    status order_status,
    changed_by TEXT,
    changed_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    note TEXT
);
