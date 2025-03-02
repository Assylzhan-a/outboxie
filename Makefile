.PHONY: test build infra-up infra-down docker-build docker-run docker-run-distributed docker-stop docker-stop-distributed docker-clean demo test-with-infra

# Complete demo for reviewers - runs the distributed example from scratch
demo:
	@./demo.sh

# Run tests
test:
	go test -v ./...

# Build the example application
build:
	go build -o bin/example ./cmd/example
	go build -o bin/subscriber ./cmd/subscriber

# Start the infrastructure (PostgreSQL and NATS)
infra-up:
	docker-compose up -d
	@echo "Waiting for services to be ready..."
	@sleep 5

# Stop the infrastructure
infra-down:
	docker-compose down

# Start infrastructure and run tests immediately
test-with-infra: infra-up
	go test -v ./...
	@echo "Tests completed. Infrastructure is still running."
	@echo "Run 'make infra-down' when you're done to stop the infrastructure."

# Build Docker images
docker-build:
	docker-compose build

# Run the application in Docker
docker-run: docker-build
	docker-compose up -d
	@echo "Application is running at http://localhost:8080"
	@echo "To view logs, run: docker-compose logs -f"

# Run the application in distributed mode in Docker
docker-run-distributed: docker-build
	docker-compose -f docker-compose.distributed.yml up -d
	@echo "Applications are running at:"
	@echo "  - http://localhost:8081 (instance1)"
	@echo "  - http://localhost:8082 (instance2)"
	@echo "  - http://localhost:8083 (instance3)"
	@echo "To view logs, run: docker-compose -f docker-compose.distributed.yml logs -f"

# Stop Docker containers
docker-stop:
	docker-compose down

# Stop distributed Docker containers
docker-stop-distributed:
	docker-compose -f docker-compose.distributed.yml down

# Clean Docker resources
docker-clean:
	docker-compose down --rmi local --volumes --remove-orphans
	docker-compose -f docker-compose.distributed.yml down --rmi local --volumes --remove-orphans 