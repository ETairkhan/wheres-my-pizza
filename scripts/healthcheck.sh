#!/bin/bash

# Health check script for restaurant system

check_postgres() {
    echo "Checking PostgreSQL..."
    if psql postgres://restaurant_user:restaurant_pass@localhost:5432/restaurant_db -c "SELECT 1" >/dev/null 2>&1; then
        echo "✓ PostgreSQL is healthy"
        return 0
    else
        echo "✗ PostgreSQL is not healthy"
        return 1
    fi
}

check_rabbitmq() {
    echo "Checking RabbitMQ..."
    if nc -z localhost 5672; then
        echo "✓ RabbitMQ is healthy"
        return 0
    else
        echo "✗ RabbitMQ is not healthy"
        return 1
    fi
}

check_tables() {
    echo "Checking database tables..."
    tables=$(psql postgres://restaurant_user:restaurant_pass@localhost:5432/restaurant_db -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public'")
    if [ "$tables" -ge 4 ]; then
        echo "✓ All tables exist ($tables tables found)"
        return 0
    else
        echo "✗ Missing tables (only $tables tables found)"
        return 1
    fi
}

check_postgres && check_rabbitmq && check_tables

if [ $? -eq 0 ]; then
    echo ""
    echo "✓ All systems are healthy!"
    exit 0
else
    echo ""
    echo "✗ Some systems are not healthy!"
    exit 1
fi