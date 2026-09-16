
![CI](https://github.com/p-sokolov/my-calendar/actions/workflows/ci.yml/badge.svg)

# Currency Quote Service

An asynchronous Go service that refreshes and serves exchange-rate quotes for
currency pairs made of USD, EUR, and MXN. A client requests a refresh, receives
an update identifier immediately, and can later retrieve either the task status
or the latest successfully stored quote.

Project demo video: https://www.dropbox.com/scl/fi/w1iyojos3wbe4zehrss2n/plata-quote-service-demo.mp4?rlkey=euy5pd2ehqqip0e9ziffeh0ad&st=um5mqtey&dl=0

## Task scope and expectations

This project represents itself:

- HTTP JSON API for currency quotes;
- update quotes through an external FX provider in the background;
- persist tasks and successful quotes in PostgreSQL;
- support USD, EUR, and MXN pairs;
- provide Docker-based local execution and OpenAPI documentation.

The service deliberately treats a quote refresh as asynchronous work. Calling a
remote provider can be slow or fail temporarily, so the HTTP request only queues
the task and returns `202 Accepted`.

### Endpoints

| Endpoint | Behaviour |
| --- | --- |
| `POST /api/v1/quotes/refresh` | Queues a quote refresh and returns `update_id`. Requires `Idempotency-Key`. |
| `GET /api/v1/quotes/updates/{id}` | Returns the state and result of one update task. |
| `GET /api/v1/quotes/latest?pair=EUR/MXN` | Returns the most recent successful quote for a supported pair. |

Supported currencies are `USD`, `EUR`, and `MXN`; every ordered pair inside this
set is accepted. A request for an unsupported pair returns `400`.

## Technology stack

- **Go, Echo, and an OpenAPI** 3.0 contract.
- **PostgreSQL** for durable tasks, idempotency records, and quote history.
- **Redis** cache-aside caching for the latest successful quote.
- Background worker with retry, exponential backoff, full jitter, and leases.
- **Docker Compose** for a production-like stack and a development stack with Air.
- `sqlc` for typed PostgreSQL queries and `oapi-codegen` for transport code.
- Unit tests for handlers, services, the worker, the FX client, and Redis cache.

## How to run

1. Create a local environment file:

```shell
cp .env.example .env
```

2. Set `FX_ACCESS_KEY` in `.env` (or take an existing api key which I used: `7e6c012328a0b46bef65b9636b81d70e`). 

3. Start the application, PostgreSQL, and Redis:

```shell
task up
```

The service starts at `http://localhost:8080`.

Useful URLs:

- Swagger UI: `http://localhost:8080/api/v1/swagger/`
- OpenAPI document: `http://localhost:8080/api/v1/openapi.json`

If you need UI for database and redis, use command below. Some prerequisites you can find in .env.example file.

```shell
task tools:up
```

By the way, Taskfile provides commands such as:

- task test
- task logs
- task down
- task tools:down

and others.

## Postman demo

Import [Quote-Service.postman_collection.json](postman/Quote-Service.postman_collection.json) into Postman or try [my shared link](https://www.postman.com/89214372442az-113996/plata-home-assignment/collection/80ku252/quote-service?action=share&source=copy-link&creator=55962463).

## Architecture

The project is implemented using Clean Architecture (handler -> service -> repository) and follows the standard Go project structure (more or less). 

### Project structure

```text
api/
  openapi.yaml                         OpenAPI contract

cmd/app/
  main.go                              Application entry point and graceful shutdown

env/
  migrate.go                           Goose migrations runner
  migrations/                          PostgreSQL schema migrations
  compose.yaml                         Production-like Docker Compose stack
  compose-dev.yaml                     Development stack with Air live reload

internal/app/
  app.go                               App wiring: configuration, DB, migrations, HTTP, handlers, worker

internal/cache/
  quotes/
    redis.go                           Redis cache-aside implementation for latest quotes

internal/client/
  fx/
    client.go                          FX client types and constructor
    exchangerates.go                   External FX provider HTTP adapter

internal/config/
  config.go                            Reads .env / environment variables

internal/errorz/
  general.go                           Shared application errors
  quotes.go                            Quote-specific business errors

internal/models/
  quotes.go                            Domain models for quote updates, cache, idempotency, and outbox

internal/repository/
  transactor.go                        PostgreSQL transaction helper
  postgres/
    postgres.go                        PostgreSQL pool helper
    sqlc/
      queries/                         SQL queries used by sqlc
      storage/                         Generated sqlc code
  quotes/
    refresh.go                         Idempotent refresh-task creation
    get.go                             Quote update and latest-quote lookups
    worker.go                          Worker claim, retry, success, and failure persistence
    repo.go                            Repository constructor and sqlc-to-domain mappers

internal/service/
  quotes/
    refresh.go                         Refresh use case and pair validation
    get.go                             Latest quote cache-aside use case
    service.go                         Service dependencies and constructor

internal/transport/http/
  middleware/
    errors.go                          Strict-handler error mapping
  v1/
    api.gen.go                         Generated OpenAPI Echo server and types
    quotes/                            HTTP handlers and response mapping

internal/worker/
  quotes/
    worker.go                          Background quote processing, lease handling, retries, and jitter

pkg/swagger-ui/                        Embedded Swagger UI assets

postman/
  Quote-Service.postman_collection.json  Numbered end-to-end API demo collection

tests/                                 Unit tests for handlers, services, worker, FX client, and Redis cache
```

## Solution notes

### Idempotency

`Idempotency-Key` makes retrying `POST /quotes/refresh` safe. The service stores
a SHA-256 request hash with the returned task ID. The same key and body return
the original ID; the same key with a different body returns `409 Conflict`.
An advisory PostgreSQL lock serializes concurrent requests for one key.

### Worker, leases, and distributed coordination

Workers claim pending rows using `FOR UPDATE SKIP LOCKED`, which lets multiple
application instances consume the queue without selecting the same row. Claiming
a task sets `PROCESSING`, increments `attempt_count`, assigns `locked_until`,
and generates a `lease_token`.

The lease token is compared when a worker marks a task successful, failed, or
ready for retry. A slow or crashed worker therefore cannot overwrite the result
of another worker that reclaimed an expired task. Expired leases are returned to
the pending queue. This is a database-backed lease mechanism; it avoids a
separate distributed-lock service for this workload.

Provider failures are retried up to `WORKER_MAX_ATTEMPTS`. The delay uses capped
exponential backoff with full jitter, reducing synchronized retry spikes when an
upstream provider is unavailable.

### FX client and quote precision

The FX client is an adapter around the external provider's conversion endpoint.
It receives a pair such as `EUR/MXN`, requests one unit of the base currency,
and returns the resulting rate to the worker.

The API exposes a JSON number and PostgreSQL stores it as `NUMERIC(18,8)`. 
This level of precision is entirely sufficient for processing quote values, given that the consumer can round the value to two decimal places.

### Redis cache

`GET /quotes/latest` uses cache-aside behaviour: it reads Redis first, falls
back to PostgreSQL on a miss, then caches a successful result with a TTL. After
a worker stores a newer successful quote, it invalidates the corresponding cache
key. Redis failures do not prevent the database-backed API from working, the service can continue to function without this caching mechanism.

### OpenAPI generation

`api/openapi.yaml` is the source contract. `oapi-codegen` generates the Echo
server interfaces, strict handler types, models, and embedded OpenAPI document
in `internal/transport/http/v1/api.gen.go`.

This project adheres to the **API-First** development methodology. Instead of writing code first and generating documentation from it, the API design is treated as a foundational step. 

### Outbox

The `outbox_events` table and its pending-event index are present as a foundation
for reliable integration events. It is not active yet: the current service does
not insert events or run an outbox publisher. This does not affect the quote API;
if external side effects are added later, write the domain change and outbox row
in one transaction, then publish pending events from a separate worker.

## Production follow-ups

Natural next production improvements are a health/readiness endpoint, metrics and tracing, a separate migration job during deployment, provider error classification (retryable vs permanent), and an outbox publisher if downstream event delivery is required.

For an Internet-facing deployment, also add explicit validation for every
OpenAPI string constraint, request rate limiting, and authentication or an API
gateway policy as appropriate. The generated Echo transport enforces required
parameters and parses types, but OpenAPI `minLength` and `pattern` constraints
are not a substitute for runtime validation by themselves.


