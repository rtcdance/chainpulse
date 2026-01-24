.PHONY: help build up down logs ps clean rebuild

# Default target
help:
	@echo "ChainPulse Docker Commands"
	@echo "=========================="
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build       - Build Docker images"
	@echo "  up          - Start all services"
	@echo "  down        - Stop all services"
	@echo "  logs        - View service logs"
	@echo "  ps          - Show running services"
	@echo "  clean       - Remove containers and volumes"
	@echo "  rebuild     - Rebuild images and restart services"
	@echo "  shell       - Open shell in chainpulse container"
	@echo "  db-shell    - Open PostgreSQL shell"
	@echo "  redis-cli   - Open Redis CLI"
	@echo "  kafka-cli   - Open Kafka CLI"
	@echo ""

# Build Docker images
build:
	@echo "Building Docker images..."
	docker-compose -f docker-compose.yml build

# Start all services
up:
	@echo "Starting services..."
	docker-compose -f docker-compose.yml up -d
	@echo "Services started. Waiting for health checks..."
	@sleep 5
	@docker-compose -f docker-compose.yml ps

# Stop all services
down:
	@echo "Stopping services..."
	docker-compose -f docker-compose.yml down

# View logs
logs:
	docker-compose -f docker-compose.yml logs -f

# Show running services
ps:
	docker-compose -f docker-compose.yml ps

# Clean up containers and volumes
clean:
	@echo "Cleaning up containers and volumes..."
	docker-compose -f docker-compose.yml down -v
	@echo "Cleanup complete"

# Rebuild and restart
rebuild: clean build up
	@echo "Rebuild complete"

# Open shell in chainpulse container
shell:
	docker-compose -f docker-compose.yml exec chainpulse-monolithic sh

# Open PostgreSQL shell
db-shell:
	docker-compose -f docker-compose.yml exec postgres psql -U chainpulse -d chainpulse

# Open Redis CLI
redis-cli:
	docker-compose -f docker-compose.yml exec redis redis-cli

# Open Kafka CLI
kafka-cli:
	docker-compose -f docker-compose.yml exec kafka bash

# Health check
health:
	@echo "Checking service health..."
	@docker-compose -f docker-compose.yml ps
	@echo ""
	@echo "Testing endpoints..."
	@curl -s http://localhost:8080/health | jq . || echo "ChainPulse API not responding"
	@redis-cli -h localhost ping || echo "Redis not responding"
	@pg_isready -h localhost -U chainpulse || echo "PostgreSQL not responding"

# View metrics
metrics:
	@echo "Opening metrics dashboard..."
	@open http://localhost:8081/metrics || xdg-open http://localhost:8081/metrics || echo "Visit http://localhost:8081/metrics"

# View API documentation
api-docs:
	@echo "Opening API documentation..."
	@open http://localhost:8080/api/docs || xdg-open http://localhost:8080/api/docs || echo "Visit http://localhost:8080/api/docs"
