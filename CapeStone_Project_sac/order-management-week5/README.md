# Order Management API

A Go-based order management service with REST, GraphQL, database persistence, and lightweight Prometheus-style metrics compatible with local development environments.

## Features

- Create, list, fetch, and update order status
- PostgreSQL support with in-memory fallback for local testing
- Health endpoints for liveness and readiness
- Request ID, recovery, API key, and security headers middleware
- Lightweight metrics endpoint at `/metrics` in a Prometheus-compatible format
- GraphQL endpoint at `/graphql`

## Run locally

1. Copy `.env.example` to `.env` and adjust values if needed.
2. Start PostgreSQL with Docker Compose:
   ```bash
   docker compose up -d postgres
   ```
3. Run the API:
   ```bash
   go run .
   ```

## Docker

```bash
docker build -t order-management .
docker run --env-file .env -p 8080:8080 order-management
```

## Metrics

The service exposes a Prometheus-styled metrics endpoint at `/metrics` without requiring local Prometheus or Grafana. The endpoint returns text metrics like:

```text
http_requests_total{method="GET",path="/api/v1/orders",status="200"} 12
http_request_duration_seconds_sum{method="GET",path="/api/v1/orders"} 0.023123
http_errors_total 0
```

## Health checks

- `GET /health/live`
- `GET /health/ready`
