# Currency exchange rates microservice

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

The API listens on `HTTP_PORT` (8088 by default - you can set it in `.env`).

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
- **Response cache is process-local.** A 5-minute in-memory cache per API
  process. There's no invalidation on fetch, so a cached response can lag
  behind the DB by up to 5 minutes. Redis (or a shorter TTL, or no cache at
  all) would be the next step.
- **No retries / backoff on the upstream RSS fetch.** A flaky network turns
  into a failed currency for that run. `job.Run` already tolerates partial
  failures, so it's not catastrophic, but a single retry would help.
- **No singleflight on the response cache.** A burst of cold-cache requests
  all hit the DB. Acceptable at this traffic level; 
- **History `limit` is silently clamped to 1000 in the store.** The validator
  only checks `>= 1`. Either reject `> 1000` in the validator or document the
  cap in the response.
- **No `/healthz` or `/readyz`.** Fine for a test task; would be expected in k8s.
- **Integration tests.** Only unit tests. A real DB test would use
  testcontainers or a dedicated `make test-integration` target.
- **No structured request IDs / trace propagation.** Logs identify requests by
  method+path+timestamp only.

## Spec interpretation note

The spec says "request 1 fetches info for currency GBP ... request 2 then again
fetches the endpoint and pulls its own currency info" — but the real RSS feed
returns *all* currencies in one document. The fetch job follows the spec's
intent literally: N goroutines, N independent HTTP requests, each parsing out
the one currency it cares about. In production you'd fetch once and fan out the
parse, but that's not what was asked for.
