GO_BUILD = go build
BINARIES = ./bin
COMMANDS = ./cmd

# Go Binaries
.PHONY: bin atlas
bin:
	$(GO_BUILD) -o $(BINARIES)/ $(COMMANDS)/...

atlas:
	$(GO_BUILD) -o $(BINARIES)/atlas $(COMMANDS)/atlas

hades:
	$(GO_BUILD) -o $(BINARIES)/hades $(COMMANDS)/hades

# Docker
DOCKER_COMPOSE = docker-compose -f docker/docker-compose.yml --env-file ./.env
.PHONY: up down postgres prometheus

up:
	$(DOCKER_COMPOSE) up -d 

down:
	$(DOCKER_COMPOSE) down

postgres:
	$(DOCKER_COMPOSE) up -d postgres

prometheus:
	$(DOCKER_COMPOSE) up -d prometheus
	