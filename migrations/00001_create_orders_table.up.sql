CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TYPE order_status AS ENUM ('received', 'cooking', 'ready', 'completed', 'cancelled');
CREATE TYPE order_type AS ENUM ('dine_in', 'takeout', 'delivery');

CREATE TABLE IF NOT EXISTS orders (
    order_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    order_number TEXT UNIQUE NOT NULL,
    customer_name TEXT NOT NULL,
    type order_type NOT NULL,
    table_number INTEGER,
    delivery_address TEXT,
    total_amount NUMERIC(10, 2) NOT NULL,
    priority INTEGER DEFAULT 1,
    status order_status DEFAULT 'received',
    processed_by TEXT,
    completed_at TIMESTAMPTZ
);
