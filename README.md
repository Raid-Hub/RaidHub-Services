# RaidHub Services

This repo contains the main services RaidHub uses to populate our database and maintain our data platform.

## Pre-reqs
You will need to install docker desktop to start the service stack

### Build

- `make` - Compile binaries
- `make services` - Spin up the services stack

### Run

- `bin/atlas` - Run the PGCR crawler
- `bin/hades` - Run the missed PGCR collector
- `bin/hermes` - Run the message queue worker
- `bin/athena` - Download manifest definitions

## Migrations
- `bin/migrate` - Migrate your local database
