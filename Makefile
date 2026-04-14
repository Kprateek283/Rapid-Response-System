.PHONY: setup up down clean logs

# Copies the example env file if the real one doesn't exist yet
setup:
	@echo "Setting up local environment..."
	@if [ ! -f infrastructure/.env ]; then cp infrastructure/.env.example infrastructure/.env; echo "Created .env from .env.example"; else echo ".env already exists"; fi

# Starts all infrastructure containers in the background
up: setup
	@echo "Starting infrastructure..."
	cd infrastructure && docker compose up -d

# Stops all infrastructure containers
down:
	@echo "Stopping infrastructure..."
	cd infrastructure && docker compose down

# Stops containers AND wipes all databases/queues clean
clean:
	@echo "Nuking infrastructure and wiping volumes..."
	cd infrastructure && docker compose down -v

# Tails the logs for all infrastructure containers
logs:
	cd infrastructure && docker compose logs -f
