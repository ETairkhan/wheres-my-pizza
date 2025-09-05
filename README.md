# wheres-my-pizza

# Restaurant Order Management System

A distributed system for managing restaurant orders with microservices architecture.

## Architecture

The system consists of four main services:

1. **Order Service**: REST API for creating orders
2. **Kitchen Worker**: Processes orders from the queue
3. **Tracking Service**: API for tracking order status
4. **Notification Subscriber**: Sends notifications for status changes

## Setup

1. Install dependencies: `go mod download`
2. Run setup script: `./scripts/setup.sh`
3. Build the application: `go build -o restaurant-system .`

## Running Services

Run each service in a separate terminal:

```bash
# Order Service (port 3000)
./restaurant-system --mode=order-service

# Kitchen Worker
./restaurant-system --mode=kitchen-worker

# Tracking Service (port 3001)
./restaurant-system --mode=tracking-service

# Notification Subscriber
./restaurant-system --mode=notification-subscriber