# Outboxie - Transactional Outbox Pattern for Go

Outboxie is a test implementation of the Transactional Outbox pattern for reliable message publishing in distributed systems.

## For Reviewers: Quick Start

To test the distributed example from scratch, follow these simple steps:

1. **Clone the repository**:
   ```bash
   git clone https://github.com/assylzhan-a/outboxie.git
   cd outboxie
   ```

2. **Run the tests and/or automated demo**:
   Running tests with prebuilt infra
   ```bash
   make test-with-infra
   ```

   **Run Demo using make:**
   ```bash
   make demo
   ```

   The demo will:
   - Clean up any existing containers
   - Build all necessary Docker images
   - Start the distributed setup with 3 application instances and 3 subscribers
   - Create a test order through instance 1
   - Verify all subscribers receive the message
   - Test the leader election by stopping the leader instance
   - Verify another instance takes over leadership
   - Create another order through instance 2
   - Verify all subscribers still receive messages

3. **When finished, clean up**:
   ```bash
   make docker-stop-distributed
   ```

The demo showcases the key features of the Outboxie library:
- Reliable message delivery using the Transactional Outbox pattern
- Leader election for distributed environments
- Fault tolerance when instances fail
- Pub/sub messaging with NATS

## Manual Testing Instructions

If you prefer to test the system manually without using scripts, you can follow these step-by-step instructions using only Docker and Make commands:

1. **Clean up any existing containers**:
   ```bash
   make docker-stop-distributed
   ```

2. **Build the Docker images**:
   ```bash
   docker-compose build
   ```

3. **Start the distributed setup**:
   ```bash
   docker-compose -f docker-compose.distributed.yml up -d
   ```

4. **Wait for services to initialize** (about 15 seconds for PostgreSQL and NATS to be fully ready)

5. **Create a test order through instance 1**:
   ```bash
   curl -X POST http://localhost:8081/orders \
     -H "Content-Type: application/json" \
     -d '{"customer_id": "550e8400-e29b-41d4-a716-446655440000", "amount": 100.50}'
   ```

6. **Check if all subscribers received the message**:
   ```bash
   docker-compose -f docker-compose.distributed.yml logs subscriber1 subscriber2 subscriber3 | grep "Received order created event"
   ```

7. **Test leader election by stopping the leader instance**:
   ```bash
   docker-compose -f docker-compose.distributed.yml stop app1
   ```

8. **Wait for new leader election** (about 10 seconds)

9. **Check which instance became the new leader**:
   ```bash
   docker-compose -f docker-compose.distributed.yml logs --since=15s app2 app3 | grep "leader"
   ```

10. **Create another order through instance 2**:
    ```bash
    curl -X POST http://localhost:8082/orders \
      -H "Content-Type: application/json" \
      -d '{"customer_id": "550e8400-e29b-41d4-a716-446655440000", "amount": 200.75}'
    ```

11. **Verify all subscribers still receive messages**:
    ```bash
    docker-compose -f docker-compose.distributed.yml logs --tail=20 subscriber1 subscriber2 subscriber3 | grep "Received order created event"
    ```

12. **When finished, clean up**:
    ```bash
    make docker-stop-distributed
    ```

## Features

- Guarantees at-least-once message delivery
- Preserves FIFO (First-In-First-Out) message ordering
- Works with PostgreSQL as the database
- Uses NATS as the message broker
- Suitable for distributed environments with multiple service replicas
- Does not use PostgreSQL LISTEN/NOTIFY to avoid tying up database connections
- Modular architecture with separation of concerns

## Architecture

The library consists of the following components:

1. **Config**: Centralized configuration management with builder pattern for easy customization
2. **Outbox Repository**: Stores outbox messages in the database as part of the business transaction
3. **Message Processor**: Retrieves and publishes pending messages to NATS
4. **Leader Election**: Supports database-based leader election for distributed environments
5. **Publisher**: Handles the actual publishing of messages to NATS

## Example

The repository includes a runnable example demonstrating the use of the library in a simple order processing service.

## Prerequisites

- Go 1.21 or later
- Docker and Docker Compose
- Make

## Running the Example Locally

1. Clone the repository:
   ```
   git clone https://github.com/assylzhan-a/outboxie.git
   cd outboxie
   ```

2. Start the required infrastructure (PostgreSQL and NATS):
   ```
   make infra-up
   ```

3. Run the example application (publisher):
   ```
   make run-example
   ```

4. In a separate terminal, run the subscriber service:
   ```
   make run-subscriber
   ```

5. To run tests:
   ```
   make test
   ```

6. To clean up:
   ```
   make infra-down
   ```

## Running Multiple Instances (Distributed Mode)

To demonstrate the distributed nature of the library, you can run multiple instances:

```bash
make run-distributed
```

This will start 3 instances of the application, each with a different instance ID and HTTP port:
- Instance 1: Port 8081
- Instance 2: Port 8082
- Instance 3: Port 8083

The example application uses a database-based leader election mechanism to ensure that only one instance processes messages at a time. The leader election is implemented using a database table to track which instance is the current leader. Instances periodically attempt to claim leadership, and only the leader will process outbox messages.

This approach ensures that there's no duplication of message processing while still providing fault tolerance. If the leader instance fails, another instance will automatically take over after the leader's heartbeat expires.

You can also run multiple subscribers to demonstrate the pub/sub nature of the messaging system:

```bash
make run-subscribers
```

This will start 3 subscriber instances, each with a different ID. All subscribers will receive all messages published to the topics they're subscribed to.

## Testing the Application

Once the application is running, you can create an order using the following curl command:

```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id": "550e8400-e29b-41d4-a716-446655440000", "amount": 100.50}'
```

If you're running in distributed mode, you can use ports 8081, 8082, or 8083 instead.

The subscriber(s) will receive and log the order created event.