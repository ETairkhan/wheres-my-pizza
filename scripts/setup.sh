#!/bin/bash

# Setup script for restaurant system

echo "Setting up restaurant system..."

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "Docker is not installed. Please install Docker first."
    exit 1
fi

# Start PostgreSQL container
echo "Starting PostgreSQL..."
docker run --name restaurant-postgres -e POSTGRES_USER=restaurant_user -e POSTGRES_PASSWORD=restaurant_pass -e POSTGRES_DB=restaurant_db -p 5432:5432 -d postgres:15

# Start RabbitMQ container
echo "Starting RabbitMQ..."
docker run --name restaurant-rabbitmq -e RABBITMQ_DEFAULT_USER=guest -e RABBITMQ_DEFAULT_PASS=guest -p 5672:5672 -p 15672:15672 -d rabbitmq:3-management

# Wait for containers to start
echo "Waiting for containers to start..."
sleep 10

# Run database migrations
echo "Running database migrations..."
psql postgres://restaurant_user:restaurant_pass@localhost:5432/restaurant_db -f migrations/001_create_tables.sql

echo "Setup complete!"
echo "PostgreSQL is running on localhost:5432"
echo "RabbitMQ is running on localhost:5672"
echo "RabbitMQ Management UI: http://localhost:15672 (guest/guest)"