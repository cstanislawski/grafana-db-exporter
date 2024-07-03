#! /bin/bash

set -e

# cleanup previous run
# docker compose -f ./docker-compose.yml down --volumes || true
# docker volume rm "$(docker volume ls -qf dangling=true)" || true

# start
docker compose -f ./docker-compose.yml up --build --force-recreate
