#!/bin/bash

echo "Testing Restaurant System..."

# Wait for services to be ready
echo "Waiting for services to be ready..."
sleep 5

# Test if PostgreSQL is accessible using docker exec
echo "Testing PostgreSQL connection..."
if docker exec restaurant-postgres psql -U restaurant_user -d restaurant_db -c "SELECT 1" >/dev/null 2>&1; then
    echo "✓ PostgreSQL connection successful!"
else
    echo "✗ PostgreSQL is not accessible. Please check if it's running."
    exit 1
fi

# Test if RabbitMQ is accessible
echo "Testing RabbitMQ connection..."
if nc -z localhost 5672; then
    echo "✓ RabbitMQ connection successful!"
else
    echo "✗ RabbitMQ is not accessible. Please check if it's running."
    exit 1
fi

# Test database schema using docker exec
echo "Testing database schema..."
tables=$(docker exec restaurant-postgres psql -U restaurant_user -d restaurant_db -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public'")
if [ "$tables" -ge 4 ]; then
    echo "✓ All tables exist ($tables tables found)"
else
    echo "✗ Missing tables (only $tables tables found)"
    exit 1
fi

echo "All tests passed! The system is ready to use."
echo ""
echo "To start the services, run:"
echo "  ./wheres-my-pizza --mode=order-service"
echo "  ./wheres-my-pizza --mode=kitchen-worker --worker-name=chef_john"
echo "  ./wheres-my-pizza --mode=tracking-service"
echo "  ./wheres-my-pizza --mode=notification-subscriber"