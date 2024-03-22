GO_BUILD = go build
BINARIES = ./bin
COMMANDS = ./cmd
DOCKER_COMPOSE = docker-compose

.PHONY: bin atlas
bin:
	$(GO_BUILD) -o $(BINARIES)/ $(COMMANDS)/...

atlas:
	$(GO_BUILD) -o $(BINARIES)/atlas $(COMMANDS)/atlas

hades:
	$(GO_BUILD) -o $(BINARIES)/hades $(COMMANDS)/hades

.PHONY: postgres
postgres:
	$(DOCKER_COMPOSE) up -d postgres