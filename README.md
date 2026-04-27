# Currency exchange rates microservice

<img width="599" alt="Screenshot 2026-04-27 234146" src="https://github.com/user-attachments/assets/63d6a140-5dd6-41b7-a58b-0cab3917b065" />
<img width="599" alt="Screenshot 2026-04-27 234214" src="https://github.com/user-attachments/assets/df392e1b-8e56-4fc7-b90a-0fc81e823850" />


A small Go service that fetches currency exchange rates from the Bank of Latvia
RSS feed (`https://www.bank.lv/vk/ecb_rss.xml`), stores them in MariaDB, and
exposes them over HTTP.

There are two binaries:

- `cmd/api`   — HTTP server with two endpoints.
- `cmd/fetch` — one-shot CLI that fetches the configured currencies and writes
  them to the database. Each currency is fetched in its own goroutine, doing a
  separate HTTP request, so the fetcher behaves as if the upstream returned a
  single currency per call (per the spec).

## Endpoints

- `GET /v1/rates/latest`
  Latest rate per currency from the database.

- `GET /v1/rates/history/{currency}?limit=&order=`
  History for a single currency. `limit` is optional (default 100, hard-capped
  at 1000 in the store). `order` is `asc` or `desc` (default `desc`).
  Validation errors return `422 Unprocessable Entity` with
  `{"errors": {"field": "message"}}`.

Both endpoints set `Cache-Control: public, max-age=300` and use a 5-minute
in-process response cache (see "Skipped / could be improved" below).

## Running it

Requires Docker. The Makefile drives everything.

```
make run     # bring up db, run migrations, build & start api (idempotent)
make fetch   # run the fetch command once, save rates to the db
make logs    # tail the api logs
make test    # go test ./...
make down    # stop containers (keeps the db volume)
make clean   # stop containers and drop the db volume
```

`make run` copies `.env.local.example` to `.env` if `.env` is missing, then
`docker compose down && docker compose up -d --build db migrate api`. Safe to
re-run any time.

`make fetch` accepts arguments via `ARGS`:

```
make fetch ARGS="--currencies USD,GBP,EUR"
make fetch ARGS="--rss-url https://example.com/feed.xml --currencies USD"
```

Defaults come from `FETCH_CURRENCIES` and `RSS_URL` in `.env`.

The API listens on `HTTP_PORT` (8080 by default - you can set it in `.env`).

## Layout

```
cmd/
  api/    HTTP server entrypoint
  fetch/  CLI entrypoint
internal/
  application/  wires config -> deps -> app structs (api.go, fetch.go, bootstrap.go)
  cache/        Cache[V] interface + InMemoryCache[V]
  config/       env-var loading, split into APIConfig / FetchConfig
  data/         Store interface + MySQL implementation
  fetcher/      Transport, RatesParser, HTTPFetcher (Fetcher interface)
  job/          fan-out fetch loop with per-currency timeout
  server/       routes, handlers, middleware, helpers
  validator/    thin Validator + In/Matches predicates
migrations/     golang-migrate SQL files
```

The fetch interface is the seam from the spec: anything implementing
`fetcher.Fetcher` (`FetchOne(ctx, currency) (Rate, error)`) can be plugged into
the job loop.

## Configuration

All config is read from environment variables. See `.env.local.example` for the
full list. The fetch command also accepts `--rss-url` and `--currencies` flags
that override the env values.

## Logging

`slog` JSON to stdout. Level is set by `LOG_LEVEL` (`debug`, `info`, `warn`,
`error`). At `debug` you'll see per-currency fetch start/done, cache
hit/miss/set, and validation failures. At `info` you get one line per HTTP
request and one summary line per fetch run.

## Skipped / could be improved

Things left out to keep the test task focused. Each one is a known
gap, not an oversight:

- **Config validation.** Env vars are read with `os.Getenv` and `strconv.Atoi`
  errors are ignored. Bad values silently become zero. Fail-fast validation
  belongs in `internal/config/config.go` 
- **Response cache is probably overkill.** The API caches JSON responses in
  memory for 5 minutes. The fetch command can't invalidate it, so the API can
  serve stale data for up to 5 minutes after a fetch. Honestly, the queries
  are cheap and rates change once a day, so the cache isn't earning its keep.
  Easy fix: drop it and rely on the `Cache-Control` header for browser/CDN
  caching. If you want burst protection, drop the TTL to ~30s. For multiple
  replicas, move to Redis with explicit invalidation.

  I added the cache mostly to show how it would be wired in (the `Cache[V]`
  interface + `InMemoryCache` are easy to swap for Redis if it ever mattered).
  *For this specific use case  daily rates, indexed reads  it's overkill.*
- **No retries / backoff on the upstream RSS fetch.** A flaky network turns
  into a failed currency for that run. `job.Run` already tolerates partial
  failures, so it's not catastrophic, but a single retry would help.
- **No singleflight on the response cache.** A burst of cold-cache requests
  all hit the DB. Acceptable at this traffic level; 
- **History `limit` is silently clamped to 1000 in the store.** The validator
  only checks `>= 1`. Either reject `> 1000` in the validator or document the
  cap in the response.
- **No `/healthz` or `/readyz`.** Fine for a test task; would be expected in k8s.
- **No integration tests against real MariaDB.** The unit tests cover the
  parser, validator, handler, and fan-out job, but `MySQLStore` is only
  verified by reading the SQL. The minimum I'd add is one test under a
  `//go:build integration` tag and a `make test-integration` target, using
  testcontainers-go
- **No structured request IDs / trace propagation.** Logs identify requests by
  method+path+timestamp only.

## Spec interpretation note

The spec says "request 1 fetches info for currency GBP ... request 2 then again
fetches the endpoint and pulls its own currency info" — but the real RSS feed
returns *all* currencies in one document. The fetch job follows the spec's
intent literally: N goroutines, N independent HTTP requests, each parsing out
the one currency it cares about. In production you'd fetch once and fan out the
parse, but that's not what was asked for.
