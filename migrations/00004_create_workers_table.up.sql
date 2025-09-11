CREATE TYPE worker_status AS ENUM ('online', 'offline');

CREATE TABLE IF NOT EXISTS workers (
    worker_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    name TEXT UNIQUE NOT NULL,
    type TEXT NOT NULL,
    status worker_status DEFAULT 'online',
    last_active_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    total_orders_processed INTEGER DEFAULT 0
);
