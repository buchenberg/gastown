#!/bin/sh
set -e

COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.azure.yml}"
COMPOSE_ARGS="-f ${COMPOSE_FILE}"
if [ -f .env.azure ]; then
    COMPOSE_ARGS="${COMPOSE_ARGS} --env-file .env.azure"
fi

echo "→ Starting container (if not running)..."
docker compose ${COMPOSE_ARGS} up -d

echo "→ Attaching mayor..."
docker compose ${COMPOSE_ARGS} exec gastown gt mayor attach
