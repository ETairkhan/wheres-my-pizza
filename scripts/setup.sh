#!/bin/bash

# Setup script for restaurant system

echo "Setting up restaurant system..."

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "Docker is not installed. Please install Docker first."
    exit 1
fi

# Stop and remove existing containers if they exist
echo "Cleaning up existing containers..."
docker stop restaurant-postgres 2>/dev/null || true
docker rm restaurant-postgres 2>/dev/null || true
docker stop restaurant-rabbitmq 2>/dev/null || true
docker rm restaurant-rabbitmq 2>/dev/null || true

# Start PostgreSQL container with proper user configuration
echo "Starting PostgreSQL..."
docker run --name restaurant-postgres \
  -e POSTGRES_USER=restaurant_user \
  -e POSTGRES_PASSWORD=restaurant_pass \
  -e POSTGRES_DB=restaurant_db \
  -p 5432:5432 \
  -d postgres:15

# Start RabbitMQ container
echo "Starting RabbitMQ..."
docker run --name restaurant-rabbitmq \
  -e RABBITMQ_DEFAULT_USER=guest \
  -e RABBITMQ_DEFAULT_PASS=guest \
  -p 5672:5672 \
  -p 15672:15672 \
  -d rabbitmq:3-management

# Wait for containers to start
echo "Waiting for containers to start..."
sleep 10

# Wait specifically for PostgreSQL to be ready
echo "Waiting for PostgreSQL to be ready..."
until docker exec restaurant-postgres pg_isready -U restaurant_user -d restaurant_db; do
  sleep 2
done

# Run database migrations
echo "Running database migrations..."
# Copy the migration file to the container
docker cp migrations/001_create_tables.sql restaurant-postgres:/tmp/migration.sql

# Execute the migration
docker exec restaurant-postgres psql -U restaurant_user -d restaurant_db -f /tmp/migration.sql

echo "Setup complete!"
echo "PostgreSQL is running on localhost:5432"
echo "  Database: restaurant_db"
echo "  User: restaurant_user"
echo "  Password: restaurant_pass"
echo ""
echo "RabbitMQ is running on localhost:5672"
echo "RabbitMQ Management UI: http://localhost:15672 (guest/guest)"