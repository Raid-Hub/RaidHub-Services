default: bin

# Docker Services
DOCKER_COMPOSE = docker-compose -f docker/docker-compose.yml --env-file ./.env

up services:
	$(DOCKER_COMPOSE) up -d postgres rabbitmq

down:
	$(DOCKER_COMPOSE) down

# Single service
postgres:
	$(DOCKER_COMPOSE) up -d postgres

rabbit:
	$(DOCKER_COMPOSE) up -d rabbitmq

## Optional services
prometheus:
	$(DOCKER_COMPOSE) up -d prometheus
	
clickhouse:
	$(DOCKER_COMPOSE) up -d clickhouse


# Go Binaries
GO_BUILD = go build
BINARIES = -o ./bin/
COMMANDS = ./cmd/

.PHONY: bin
bin:
	$(GO_BUILD) $(BINARIES) $(COMMANDS)...

# Build a specific cmd with make <name>
.DEFAULT:
	$(GO_BUILD) $(BINARIES)$@ $(COMMANDS)$@
	