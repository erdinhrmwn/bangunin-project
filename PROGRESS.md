# PROGRESS — Building Materials Marketplace

FR & AC checklist per phase. Check off only after verification passes (build/test/manual). See docs/requirements.md for full detail on each item.

---

## PHASE 1 — Foundation

### Functional Requirements
- [x] FR-1.1 Project structure per CLAUDE.md §3
- [x] FR-1.2 Config (Viper + env override, fail-fast)
- [x] FR-1.3 Database (pgxpool + golang-migrate)
- [x] FR-1.4 Initial migrations (roles, users)
- [x] FR-1.5 Idempotent seeder (roles + admin)
- [x] FR-1.6 Redis client + TTL helper
- [x] FR-1.7 Logger (zerolog JSON + request log middleware)
- [x] FR-1.8 Basic middleware (recover, request ID, CORS, rate limit)
- [x] FR-1.9 Response & error (pkg/response, pkg/apperr)
- [x] FR-1.10 Health endpoint
- [x] FR-1.11 Worker skeleton (Asynq + system:heartbeat)
- [x] FR-1.12 Tooling (Makefile, golangci-lint, .env.example, docker-compose)
- [x] FR-1.13 Graceful shutdown

### Acceptance Criteria
- [x] AC-1.a docker compose up healthy + migrate + seed + health 200
- [x] AC-1.b Redis down → 503; panic → clean 500, process stays alive
- [x] AC-1.c lint & test green; migrate down/up clean; seed idempotent
- [x] AC-1.d Rate limit 429 at 70 req/min

---

## PHASE 2 — Auth + RBAC

### Functional Requirements
- [ ] FR-2.1 Register
- [ ] FR-2.2 Email verification (OTP)
- [ ] FR-2.3 Login (JWT + refresh + brute force guard)
- [ ] FR-2.4 Refresh (rotation)
- [ ] FR-2.5 Logout (jti blacklist)
- [ ] FR-2.6 Password reset
- [ ] FR-2.7 Auth middleware
- [ ] FR-2.8 RequireRole middleware
- [ ] FR-2.9 Basic profile (/user/me)
- [ ] FR-2.10 Notification stub (log-only)

### Acceptance Criteria
- [ ] AC-2.a Happy path register→verify→login→me→refresh→logout→old token rejected
- [ ] AC-2.b RBAC 403/401 per role
- [ ] AC-2.c Refresh token reuse rejected; password reset invalidates old sessions
- [ ] AC-2.d Unit test coverage for auth cases

---

## PHASE 3 — Supplier Onboarding

### Functional Requirements
- [ ] FR-3.1 Media module (upload + resize)
- [ ] FR-3.2 Store profile
- [ ] FR-3.3 KYC documents
- [ ] FR-3.4 Bank account
- [ ] FR-3.5 Submit application
- [ ] FR-3.6 Admin review (approve/reject/suspend)
- [ ] FR-3.7 Audit log
- [ ] FR-3.8 Notifications
- [ ] FR-3.9 RequireApprovedSupplier guard

### Acceptance Criteria
- [ ] AC-3.a Full flow profile→documents→submit→reject→fix→approve
- [ ] AC-3.b File validation (type/size) + resize worker
- [ ] AC-3.c Submit without NIB → 422; selling endpoint before approved → 403
- [ ] AC-3.d New migration up/down clean

---

## PHASE 4 — Catalog

### Functional Requirements
- [ ] FR-4.1 Category (admin CRUD + public tree + cache)
- [ ] FR-4.2 Product (supplier CRUD)
- [ ] FR-4.3 Variant
- [ ] FR-4.4 Product images
- [ ] FR-4.5 Stock & movements
- [ ] FR-4.6 Public catalog (search + filter + cursor)
- [ ] FR-4.7 Public detail (cache invalidate)
- [ ] FR-4.8 search_vector trigger + GIN index
- [ ] FR-4.9 Public store
- [ ] FR-4.10 Low-stock job

### Acceptance Criteria
- [ ] AC-4.a Published product appears & is found in search
- [ ] AC-4.b Publish validation (image, weight_gram, unique SKU)
- [ ] AC-4.c Stock adjustment (409 when negative, movements correct)
- [ ] AC-4.d Suspended supplier doesn't appear; cache invalidates fast
- [ ] AC-4.e Search integration test with testcontainers

---

## PHASE 5 — Core Transactions

### Functional Requirements
- [ ] FR-5.1 Cart
- [ ] FR-5.2 RajaOngkir client (ShippingGateway + mock mode)
- [ ] FR-5.3 Checkout preview
- [ ] FR-5.4 Checkout confirm (lock + TX + reservation)
- [ ] FR-5.5 Payment-service (services/payment, gRPC, Xendit)
- [ ] FR-5.6 Paid callback (idempotent)
- [ ] FR-5.7 order:expire job
- [ ] FR-5.8 Order state machine
- [ ] FR-5.9 Shipment
- [ ] FR-5.10 Order listing (user/supplier/admin)
- [ ] FR-5.11 order:autocomplete job

### Acceptance Criteria
- [ ] AC-5.a E2E multi-supplier checkout→paid→shipped→delivered→completed
- [ ] AC-5.b Concurrent confirm race → exactly one succeeds
- [ ] AC-5.c Expire job releases reservation; duplicate paid callback idempotent
- [ ] AC-5.d Cancel while paid → refund + stock restored; illegal transition 409
- [ ] AC-5.e Payment-service is a separate process; monolith degrades gracefully if it's down

---

## PHASE 6 — Post-Transaction

### Functional Requirements
- [ ] FR-6.1 Settlement on complete (ledger + balance)
- [ ] FR-6.2 Ledger (append-only)
- [ ] FR-6.3 Withdraw
- [ ] FR-6.4 Approval & disbursement
- [ ] FR-6.5 Review
- [ ] FR-6.6 Full notifications (real notification-service)
- [ ] FR-6.7 Lightweight notification preferences

### Acceptance Criteria
- [ ] AC-6.a Settlement ledger correct; idempotent
- [ ] AC-6.b Withdraw lifecycle + Σ ledger = balance; ledger UPDATE rejected by DB
- [ ] AC-6.c Review guard (403/409) + aggregate rating correct
- [ ] AC-6.d Notification-service down → doesn't fail the transaction (retry)

---

## PHASE 7 — Complementary & Hardening

### Functional Requirements
- [ ] FR-7.1 Wishlist
- [ ] FR-7.2 Banner
- [ ] FR-7.3 Supplier report
- [ ] FR-7.4 Admin report
- [ ] FR-7.5 Security hardening
- [ ] FR-7.6 Observability (metrics + Grafana)
- [ ] FR-7.7 Quality (coverage, CI, seed-demo)
- [ ] FR-7.8 Documentation (OpenAPI, README, final PROGRESS.md)

### Acceptance Criteria
- [ ] AC-7.a CI green + seed-demo + 3-role demo
- [ ] AC-7.b Metrics + Grafana + load test p95 < 300ms
- [ ] AC-7.c govulncheck clean; internal callback without secret → 401
- [ ] AC-7.d OpenAPI valid & 100% route coverage

---

## Decisions

Decision notes made when encountering ambiguity (filled in over time, per phase).

### Phase 1

- **Host vs container DSN**: `.env` (used for host-side `go run`) uses `localhost` for `APP_DB_DSN`/`APP_REDIS_ADDR`. The `api`/`worker` containers in `docker-compose.yml` need service-name DNS (`postgres`, `redis`), not `localhost`. Solution: explicit override via `environment:` in `docker-compose.yml` for both services (wins over `env_file`), `.env` stays as-is for direct host usage.
- AC-1.a–1.d verification ran via a separate container on the `deploy_default` network (not host `localhost:5432`) because a pre-existing local Postgres process conflicts on host port 5432. That local Postgres process was not touched.
