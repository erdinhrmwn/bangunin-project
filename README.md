# Bangunin

Vertical marketplace for building materials & construction supplies (Indonesia). Suppliers sell to contractors/builders/project owners with escrow payments, per-supplier order splitting, and weight-based shipping.

See [CLAUDE.md](CLAUDE.md) for architecture and design decisions, [docs/requirements.md](docs/requirements.md) for functional requirements, [docs/erd.md](docs/erd.md) for the data model, and [docs/openapi.yaml](docs/openapi.yaml) for the full API spec.

## Stack

Go 1.23+ · Fiber v3 · PostgreSQL 16 · Redis 7 · Asynq · gRPC (payment-service, notification-service) · RajaOngkir · Cloudflare R2 (MinIO in dev)

## Quickstart (<10 min)

```bash
git clone <repo-url> bangunin && cd bangunin
cp .env.example .env          # defaults work as-is for local dev
make docker-up                # postgres, redis, minio, payment-service, notification-service
make seed-demo                # migrate up + roles/admin + demo dataset
make run                      # API on :8080
```

In a second terminal, start the background worker (order expiry, stock release, email, low-stock cron):

```bash
make run-worker
```

Verify:

```bash
curl localhost:8080/health
```

## Demo accounts

`make seed-demo` creates 3 approved suppliers, 30 products, and 1 order per status. All demo accounts share the password `Password123!`.

| Role     | Email                          |
|----------|---------------------------------|
| Buyer    | demo-buyer@bangunin.test        |
| Supplier | demo-supplier1@bangunin.test    |
| Supplier | demo-supplier2@bangunin.test    |
| Supplier | demo-supplier3@bangunin.test    |
| Admin    | see `SEED_ADMIN_EMAIL` in `.env` |

Login: `POST /api/v1/auth/login` with `{"email": "...", "password": "Password123!"}`, use the returned access token as `Authorization: Bearer <token>`.

## Common tasks

```bash
make help          # list all targets
make test           # unit tests
make test-cover      # unit tests + coverage report
make lint            # golangci-lint
make migrate-up       # apply migrations
make migrate-down      # roll back last migration
make migrate-create name=xxx   # new migration pair
make docker-logs        # tail infra logs
make vuln             # govulncheck
```

Route-vs-spec parity check (AC-7.d):

```bash
python3 scripts/check_openapi_routes.py
```

## Project layout

Clean Architecture, `delivery → usecase → domain ← repository/infra`. See CLAUDE.md §3 for the full directory map.
