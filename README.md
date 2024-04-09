# RaidHub Services

This repo contains the main services RaidHub uses to populate our database and maintain our data.

### Atlas

This is our PGCR crawler

### Hades

This tool just processes any missed PGCRs, usually on a cron job

## Commands

### Build

- `make postgres` - Run Postgres Database in Docker
- `make atlas` - Build the Atlas PGCR crawler binary
- `make hades` - Build the Hades binary
- `make hermes` - Build the message queue worker
- `make bin` - Build all Go binaries

### Run

- `bin/atlas <workers>` - Run the PGCR crawler
- `bin/hades` - Run the missed PGCR collector
- `bin/hermes` - Run the message queue worker
