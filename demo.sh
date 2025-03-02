#!/bin/bash
set -e

echo "=== Cleaning up any existing containers ==="
docker-compose -f docker-compose.distributed.yml down 2>/dev/null || true
docker-compose down 2>/dev/null || true

echo "=== Building Docker images ==="
docker-compose build

echo -e "\n=== Starting distributed application ==="
docker-compose -f docker-compose.distributed.yml up -d

echo -e "\n=== Waiting for services to initialize (15 seconds) ==="
sleep 15

echo -e "\n=== Creating a test order through instance 1 ==="
curl -X POST http://localhost:8081/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id": "550e8400-e29b-41d4-a716-446655440000", "amount": 100.50}'

echo -e "\n\n=== Checking subscriber logs (all should receive the message) ==="
sleep 2
docker-compose -f docker-compose.distributed.yml logs --tail=20 subscriber1 subscriber2 subscriber3 | grep "Received order created event" || echo "No messages received yet, waiting longer..."
sleep 5
docker-compose -f docker-compose.distributed.yml logs --tail=20 subscriber1 subscriber2 subscriber3 | grep "Received order created event" || echo "Still no messages received. There might be an issue with the setup."

echo -e "\n=== Testing leader election: stopping the leader instance (app1) ==="
docker-compose -f docker-compose.distributed.yml stop app1

echo -e "\n=== Waiting for new leader election (10 seconds) ==="
sleep 10

echo -e "\n=== Checking which instance became the new leader ==="
docker-compose -f docker-compose.distributed.yml logs --since=15s app2 app3 | grep "leader" || echo "No leader election logs found, waiting longer..."
sleep 5
docker-compose -f docker-compose.distributed.yml logs --since=20s app2 app3 | grep "leader" || echo "No leader election logs found. There might be an issue with the leader election mechanism."

echo -e "\n=== Creating another order through instance 2 ==="
curl -X POST http://localhost:8082/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id": "550e8400-e29b-41d4-a716-446655440000", "amount": 200.75}'

echo -e "\n\n=== Checking subscriber logs again ==="
sleep 2
docker-compose -f docker-compose.distributed.yml logs --tail=10 subscriber1 subscriber2 subscriber3 | grep "Received order created event" || echo "No messages received yet, waiting longer..."
sleep 5
docker-compose -f docker-compose.distributed.yml logs --tail=10 subscriber1 subscriber2 subscriber3 | grep "Received order created event" || echo "Still no messages received. There might be an issue with the setup."

echo -e "\n=== Demo completed! ==="
echo "To stop the distributed setup: make docker-stop-distributed" 