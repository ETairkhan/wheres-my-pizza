#!/bin/bash

# Status script for restaurant system

echo "=== Restaurant System Status ==="
echo ""

# PostgreSQL status
echo "PostgreSQL:"
if docker ps | grep -q restaurant-postgres; then
    echo "  ✓ Container is running"
    if docker exec restaurant-postgres psql -U restaurant_user -d restaurant_db -c "SELECT 1" >/dev/null 2>&1; then
        echo "  ✓ Connection successful"

        # Count orders
        orders=$(docker exec restaurant-postgres psql -U restaurant_user -d restaurant_db -t -c "SELECT COUNT(*) FROM orders" | tr -d ' ')
        echo "  ✓ Orders in database: $orders"

        # Count workers
        workers=$(docker exec restaurant-postgres psql -U restaurant_user -d restaurant_db -t -c "SELECT COUNT(*) FROM workers" | tr -d ' ')
        echo "  ✓ Workers registered: $workers"
    else
        echo "  ✗ Connection failed"
    fi
else
    echo "  ✗ Container is not running"
fi
echo ""

# RabbitMQ status
echo "RabbitMQ:"
if docker ps | grep -q restaurant-rabbitmq; then
    echo "  ✓ Container is running"
    if nc -z localhost 5672; then
        echo "  ✓ Connection successful"
    else
        echo "  ✗ Connection failed"
    fi
else
    echo "  ✗ Container is not running"
fi
echo ""

# Check if services are running
echo "Microservices:"
services=("order-service" "kitchen-worker" "tracking-service" "notification-subscriber")
for service in "${services[@]}"; do
    if pgrep -f "./wheres-my-pizza --mode=$service" >/dev/null; then
        echo "  ✓ $service is running"
    else
        echo "  ✗ $service is not running"
    fi
done