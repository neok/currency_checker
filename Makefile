.PHONY: run api down logs fetch test test-integration clean

# Bring up DB, run migrations, build  start the API.
run:
	@test -f .env || cp .env.local.example .env
	docker compose down
	docker compose up -d --build db migrate api

# Start (or rebuild) only the API service.
#   make api
api:
	docker compose up -d --build api

down:
	docker compose down

logs:
	docker compose logs -f api

# Run fetch command.
#   make fetch
#   make fetch ARGS="--currencies USD,GBP,EUR"
#   make fetch ARGS="--rss-url https://example.com/feed.xml --currencies USD"
fetch:
	docker compose run --rm fetch $(ARGS)

test:
	go test ./...

# Integration tests. Spins up real MariaDB via testcontainers; needs Docker.
test-integration:
	go test -tags=integration -timeout 120s ./...

clean:
	docker compose down -v
