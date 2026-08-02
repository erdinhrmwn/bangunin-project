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
- [x] FR-2.1 Register
- [x] FR-2.2 Email verification (OTP)
- [x] FR-2.3 Login (JWT + refresh + brute force guard)
- [x] FR-2.4 Refresh (rotation)
- [x] FR-2.5 Logout (jti blacklist)
- [x] FR-2.6 Password reset
- [x] FR-2.7 Auth middleware
- [x] FR-2.8 RequireRole middleware
- [x] FR-2.9 Basic profile (/user/me)
- [x] FR-2.10 Notification stub (log-only)

### Acceptance Criteria
- [x] AC-2.a Happy path register→verify→login→me→refresh→logout→old token rejected
- [x] AC-2.b RBAC 403/401 per role
- [x] AC-2.c Refresh token reuse rejected; password reset invalidates old sessions
- [x] AC-2.d Unit test coverage for auth cases

---

## PHASE 3 — Supplier Onboarding

### Functional Requirements
- [x] FR-3.1 Media module (upload + resize)
- [x] FR-3.2 Store profile
- [x] FR-3.3 KYC documents
- [x] FR-3.4 Bank account
- [x] FR-3.5 Submit application
- [x] FR-3.6 Admin review (approve/reject/suspend)
- [x] FR-3.7 Audit log
- [x] FR-3.8 Notifications
- [x] FR-3.9 RequireApprovedSupplier guard

### Acceptance Criteria
- [x] AC-3.a Full flow profile→documents→submit→reject→fix→approve
- [x] AC-3.b File validation (type/size) + resize worker
- [x] AC-3.c Submit without NIB → 422; selling endpoint before approved → 403
- [x] AC-3.d New migration up/down clean

---

## PHASE 4 — Catalog

### Functional Requirements
- [x] FR-4.1 Category (admin CRUD + public tree + cache)
- [x] FR-4.2 Product (supplier CRUD)
- [x] FR-4.3 Variant
- [x] FR-4.4 Product images
- [x] FR-4.5 Stock & movements
- [x] FR-4.6 Public catalog (search + filter + cursor)
- [x] FR-4.7 Public detail (cache invalidate)
- [x] FR-4.8 search_vector trigger + GIN index
- [x] FR-4.9 Public store
- [x] FR-4.10 Low-stock job

### Acceptance Criteria
- [x] AC-4.a Published product appears & is found in search
- [x] AC-4.b Publish validation (image, weight_gram, unique SKU)
- [x] AC-4.c Stock adjustment (409 when negative, movements correct)
- [x] AC-4.d Suspended supplier doesn't appear; cache invalidates fast
- [x] AC-4.e Search integration test with testcontainers

---

## PHASE 5 — Core Transactions

### Functional Requirements
- [x] FR-5.1 Cart
- [x] FR-5.2 RajaOngkir client (ShippingGateway + mock mode)
- [x] FR-5.3 Checkout preview
- [x] FR-5.4 Checkout confirm (lock + TX + reservation)
- [x] FR-5.5 Payment-service (services/payment, gRPC, Xendit)
- [x] FR-5.6 Paid callback (idempotent)
- [x] FR-5.7 order:expire job
- [x] FR-5.8 Order state machine
- [x] FR-5.9 Shipment
- [x] FR-5.10 Order listing (user/supplier/admin)
- [x] FR-5.11 order:autocomplete job

### Acceptance Criteria
- [x] AC-5.a E2E multi-supplier checkout→paid→shipped→delivered→completed
- [x] AC-5.b Concurrent confirm race → exactly one succeeds
- [x] AC-5.c Expire job releases reservation; duplicate paid callback idempotent
- [x] AC-5.d Cancel while paid → refund + stock restored; illegal transition 409
- [x] AC-5.e Payment-service is a separate process; monolith degrades gracefully if it's down

---

## PHASE 6 — Post-Transaction

### Functional Requirements
- [x] FR-6.1 Settlement on complete (ledger + balance)
- [x] FR-6.2 Ledger (append-only)
- [x] FR-6.3 Withdraw
- [x] FR-6.4 Approval & disbursement
- [x] FR-6.5 Review
- [x] FR-6.6 Full notifications (real notification-service)
- [x] FR-6.7 Lightweight notification preferences

### Acceptance Criteria
- [x] AC-6.a Settlement ledger correct; idempotent
- [x] AC-6.b Withdraw lifecycle + Σ ledger = balance; ledger UPDATE rejected by DB
- [x] AC-6.c Review guard (403/409) + aggregate rating correct
- [x] AC-6.d Notification-service down → doesn't fail the transaction (retry)

---

## PHASE 7 — Complementary & Hardening

### Functional Requirements
- [x] FR-7.1 Wishlist
- [x] FR-7.2 Banner
- [x] FR-7.3 Supplier report
- [x] FR-7.4 Admin report
- [x] FR-7.5 Security hardening
- [x] FR-7.6 Observability (metrics + Grafana)
- [x] FR-7.7 Quality (coverage, CI, seed-demo)
- [x] FR-7.8 Documentation (OpenAPI, README, final PROGRESS.md)

### Acceptance Criteria
- [x] AC-7.a CI green + seed-demo + 3-role demo
- [x] AC-7.b Metrics + Grafana + load test p95 < 300ms
- [x] AC-7.c govulncheck clean; internal callback without secret → 401
- [x] AC-7.d OpenAPI valid & 100% route coverage

---

## Decisions

Decision notes made when encountering ambiguity (filled in over time, per phase).

### Phase 1

- **Host vs container DSN**: `.env` (used for host-side `go run`) uses `localhost` for `APP_DB_DSN`/`APP_REDIS_ADDR`. The `api`/`worker` containers in `docker-compose.yml` need service-name DNS (`postgres`, `redis`), not `localhost`. Solution: explicit override via `environment:` in `docker-compose.yml` for both services (wins over `env_file`), `.env` stays as-is for direct host usage.
- AC-1.a–1.d verification ran via a separate container on the `deploy_default` network (not host `localhost:5432`) because a pre-existing local Postgres process conflicts on host port 5432. That local Postgres process was not touched.

### Phase 2

- **Switched dev DB to local Postgres**: at the user's request, `.env` `APP_DB_DSN` now points at the local Postgres install (`root` user, no password, `bangunin` DB) instead of the dockerized one. Removed the `deploy-postgres-1`/`deploy-api-1`/`deploy-worker-1` containers and the `deploy_postgres_data` volume; Redis and MinIO stay on docker compose. Migrations verified clean (up → down → up) against the local DB.
- **Bug found and fixed during AC-2.c e2e verification**: `ResetPassword`'s docstring claimed it revoked existing sessions but never did — the old refresh token still worked after a password reset. Fixed by tracking each user's refresh tokens in a Redis set (`sessions:<userID>`) and adding `AuthRepository.RevokeAllRefreshTokens`, called from `ResetPassword`. Regenerated mocks, added a unit test, re-verified live via curl.

### Phase 3

- **`deploy/docker-compose.yml` still defines a `postgres` service** (Phase 2 only removed the *running containers*, not the compose service). Bringing the stack up recreates a `deploy-postgres-1` container that binds host port 5432 alongside the pre-existing local Postgres — both listeners appear in `lsof -i :5432`. `localhost`/`127.0.0.1` resolution always reaches local Postgres first, so `go run ./cmd/migrate` and host-run api/worker have always correctly targeted local Postgres; the docker `postgres` service silently sits empty and unused. This caused a false alarm during AC-3.d verification (docker container had no tables — expected, it's not the DB being migrated). Resolution: stop/remove `postgres`/`api`/`worker` containers again after any docker-compose-up check; verification runs api/worker via `go run` on host against local Postgres, matching Phase 2's setup. Left the compose service definition as-is (harmless when not started) rather than editing docker-compose.yml, since AC-1.a/AC-3 checks may still want a fully-dockerized stack option later.
- **Bug found and fixed during AC-3.a e2e verification**: `NotificationRepository.Create` and `AuditLogRepository.Create` inserted `entity.Notification.Data`/`entity.AuditLog.Metadata` directly, and pgx encodes a nil Go map as SQL `NULL`. Both `notifications.data` and `audit_logs.metadata` columns are `NOT NULL DEFAULT '{}'`, so any caller passing a nil map (e.g. `adminsupplier.Reject`/`Approve` never set `Notification.Data`) failed the insert — surfaced to the client only as a generic 500 (`apperr` deliberately doesn't leak DB errors, and only panic-recovery middleware logs error detail, so this required reading the repository source directly rather than server logs). Fixed by defaulting nil maps to `map[string]any{}` in both repositories before insert, so the fix covers all current and future callers.
- **Bug found and fixed during AC-3.d `docker compose up` verification**: `.env`'s `APP_R2_ENDPOINT=localhost:9000` doesn't resolve inside the Docker Compose network (container-internal `localhost` is the container itself, not the host or the `minio` service) — same class of issue already handled for `APP_DB_DSN`/`APP_REDIS_ADDR` via `environment:` overrides, but `APP_R2_ENDPOINT` was missed when Phase 3 added MinIO/R2 usage. Symptom: `api`/`worker` containers failed on startup with `storage: check bucket: ... dial tcp [::1]:9000: connect: connection refused`. Fixed by adding `APP_R2_ENDPOINT: minio:9000` and a `minio: condition: service_healthy` dependency to both services in `docker-compose.yml`. Re-verified: `docker compose up` brings all 5 containers healthy/running, api reports 61 handlers (not the earlier stale-image 7), no MinIO error.

### Phase 5

- **Bug found and fixed during AC-5.c verification**: `.env` was missing `APP_PAYMENT_GRPC_ADDR`/`APP_PAYMENT_INTERNAL_SECRET` entirely (present in `.env.example` since this phase added them, but never copied into the working `.env`). Symptom: `X-Internal-Secret` check on `POST /internal/payments/callback` always failed with 401 (empty configured secret never matches a real header value) — surfaced only when testing the duplicate-paid-callback idempotency path directly, since normal checkout flow doesn't call this endpoint from a test client. Fixed by adding both keys to `.env` with a dev value. Also had to correct the callback JSON field name during testing — the request DTO field is `xendit_invoice_id`, not `invoice_id`.
- **Local-environment note (not a code bug, no fix needed in-repo)**: while diagnosing the `order:autocomplete` background job for AC-5.a, all `localhost:6379`-targeted diagnostic tooling was silently connecting to an unrelated host `redis-server` process (managed by the macOS app DBngin) squatting port 6379 ahead of Docker's forwarding for `deploy-redis-1`, rather than the actual container the worker consumes from. Any future ad-hoc diagnostic/enqueue tooling against the live docker-compose stack's Redis must run inside the `deploy_default` docker network (e.g. `docker run --network deploy_default ... Addr:"redis:6379"`), never `localhost:6379` from the host, on a machine where this local port conflict exists.
- **AC-5.a–5.e verification evidence**: AC-5.a — 2-supplier checkout confirmed (2 orders, 1 checkout_group, 1 mock invoice), paid callback → both orders `paid`, processed→shipped (courier), delivered→completed via `order:autocomplete` job; both orders confirmed `status=completed` in DB. AC-5.b — concurrent `checkout/confirm` from 2 users on a stock=1 variant: one 201, one 409 CONFLICT; final stock non-negative (1, since reservation doesn't decrement until paid). AC-5.c — backdated `stock_reservations.expires_at`, `order:expire` job released the reservation (status→released) and expired the order, stock unaffected; duplicate paid-callback (same invoice, called twice) both returned 200 with no duplicate status-history rows (6 entries per order, one `paid` transition each). AC-5.d — cancelling an already-`expired` order → 409 `INVALID_STATUS_TRANSITION`; cancelling a `paid` order → stock restored (0→1), status `cancelled`, history note `"Order cancelled, refund processed (mock)"`. AC-5.e — stopped `payment-service` container, checkout confirm returned 502 `PAYMENT_UNAVAILABLE` (not 500/crash), `/health` stayed 200 throughout; container restarted cleanly. `make lint && make test` green; migrations verified up→down→up clean against a disposable throwaway Postgres container (23 tables + schema_migrations, matching on all three passes); `docker compose up` brings all 6 containers (postgres, redis, minio, api, worker, payment-service) healthy/running.

### Phase 6

- **`notifications.type` CHECK constraint bug (pre-existing, Phase 4)**: `entity.NotificationTypeLowStock = "low_stock"` was already inserted by `internal/usecase/inventory/inventory.go`'s low-stock check, but migration `000006_create_notifications.up.sql`'s CHECK constraint never included `'low_stock'` in its allowed values — any such insert failed at the DB level with no caller-visible symptom until the FR-6.6 low-stock-notification AC exercised the path. Fixed via new migration `000021_alter_notifications_add_low_stock.{up,down}.sql` (drop + recreate the CHECK constraint including `'low_stock'`). Not scope creep — bundled here because it blocks the FR-6.6 AC.
- **`PaymentGateway.Disburse` added to the existing gRPC contract** (FR-6.3/6.4): extended `payment.proto`'s `PaymentService` with a `Disburse` RPC (mirroring `CreateInvoice`'s mock-mode branch in `services/payment/xendit.go`) rather than adding a second gRPC service, since disbursement is still a Xendit-mediated payment operation and belongs on the same client/server pair.
- **`onOrderCompleted` (FR-6.1) idempotency relies on the order state machine, not a new DB mechanism**: `completed` is terminal in the `transitions` table (no outgoing entries), and both call sites (`Complete`, `HandleAutocomplete`) only invoke `onOrderCompleted` immediately after a successful `transition()` call — so a second attempt to complete an already-completed order fails with `INVALID_STATUS_TRANSITION` before settlement logic would ever run twice. No idempotency-key column or check-then-act repository method was added; the existing invariant already prevents double-settlement.
- **Withdraw approval/disbursement modeled as fully synchronous** (FR-6.4): every current `PaymentGateway.Disburse` implementation resolves inline (`grpcclient.PaymentClient.Disburse` → `services/payment`'s mock-mode `xenditClient.Disburse`, which always succeeds synchronously). `payout.Approve` therefore calls `Disburse` inline and settles the result (ledger `debit_withdraw` entry on success, balance restore + `failed` status on error) in the same request, instead of scaffolding a `processing` status plus a separate `/internal/withdraws/callback` endpoint for an async path nothing currently exercises. **Consequence for Task/sub-step "internal_handler.go withdraw callback + route"**: dropped from scope — no callback endpoint is needed while disbursement stays synchronous. Upgrade path if a real (async) Xendit disbursement integration replaces the mock: add a `processing` status, add the callback handler, move settlement there.
- **`ledger_entries.order_id` made nullable + `withdraw_request_id` added** (FR-6.3): the original FR-6.1 schema only supported order-based ledger entries. Withdraw disbursement also needs an append-only ledger row (`debit_withdraw`) referencing a `withdraw_request_id` instead. Rather than a separate ledger table, `order_id` was relaxed to nullable and a `ledger_entries_ref_check` CHECK constraint added (order-type entries require `order_id` + forbid `withdraw_request_id`, `debit_withdraw` requires the reverse) — keeps one physical ledger for reconciliation instead of splitting it per source.
- **`entity.OrderItem` has no direct `ProductID`** (FR-6.5): only `VariantID`. The review usecase resolves `order_item_id` → `product_id` via `OrderItem.VariantID` → `ProductVariantRepository.FindByID` → `ProductVariant.ProductID`, instead of adding a denormalized `ProductID` to `order_items` (avoids touching an existing snapshot table for a read-path-only need). Also added `OrderRepository.FindItemByID(ctx, itemID)` — no prior method fetched a single order item by its own ID (only `ListItemsByOrderID(orderID)` existed), needed since review creation is keyed off `order_item_id`, not `order_id`. Mirrors `CartRepository.FindItemByID`.
- **FR-6.7 notification preference is a single `users.email_marketing` boolean**, not a separate preferences table — reused the existing `User` entity/repo/`Update` path (`PATCH /user/me/notification-settings`), consistent with the plan's decision #7 (`ponytail: one bool column, promote to a preferences table if more toggles appear`).
- **Bug found and fixed during AC-6.a verification (pre-existing, Phase 5)**: the supplier `Deliver` usecase method existed since Phase 5 but had no HTTP route/handler — `order_handler.go` only exposed `ListMine, GetMine, Cancel, ListSupplier, GetSupplier, Process, Ship, ListAdmin, ForceStatus`. Left no legitimate way to move an order to `delivered` short of the `order:autocomplete` job (which requires an already-`delivered` order) or admin `ForceStatus` (which bypasses `onOrderCompleted` settlement entirely). Fixed by adding `OrderHandler.Deliver` + `POST /supplier/orders/:id/deliver` (mirrors `Ship`'s pattern). Not scope creep — bundled here because it blocked the only legitimate path to exercise FR-6.1 settlement.

### Phase 7

- **FR-7.7 coverage target met at aggregate level**: added targeted tests to `payout`, `adminsupplier`, `auth`, `banner`, `cart`, `order`, `product`, `user` usecases (delegation/branch cases previously uncovered) rather than chasing 100% per-package. Aggregate `internal/usecase/...` coverage measured at 75.5%, above the ≥70% target — stopped there per YAGNI, no further coverage-only tests added to packages already comfortably above target.
- **FR-7.7 CI integration job**: `.github/workflows/ci.yml` gained an `integration` job running `go test -tags=integration -v ./internal/repository/...`, gating `build-image`. The 3 existing `//go:build integration` test files (testcontainers-based, self-contained) needed no changes — only CI wiring was missing. Verified locally: both `internal/repository/postgres` and `internal/repository/redis` pass against ephemeral testcontainers.
- **FR-7.7 seed-demo**: added `seeds/demo.go` (`Demo(ctx, db)`) seeding 3 approved suppliers (10 products each, 1 variant per product, `weight_gram` set per CLAUDE.md §4) and 8 orders (one per `Order` status, each with its own buyer, checkout_group, and order_item). Wired via a new `-demo` flag on `cmd/migrate` (only runs when `-seed -demo` both passed, keeping the plain `make seed` fast) and a new `make seed-demo` target. Idempotent via `ON CONFLICT ... DO UPDATE SET <key> = EXCLUDED.<key> RETURNING id` (upsert-by-natural-key, not `DO NOTHING`, since child rows need the existing parent's ID on rerun) — verified by running `make seed-demo` twice against a live local Postgres: identical counts (3/30/8), exit 0 both times.
- **AC-6.a–6.d verification evidence**: AC-6.a — order driven through the full real state machine (checkout confirm→paid→process→ship→deliver→backdated delivered-history timestamp→`order:autocomplete` job) to `completed`; `ledger_entries` shows `credit_order` 130000.00 + `debit_commission` 5200.00 (4% of subtotal), `supplier_balances.balance` = 124800.00. AC-6.b — direct `UPDATE`/`DELETE` on `ledger_entries` both rejected by the append-only trigger; withdraw request (50000) debits balance immediately (124800→74800), admin approve → synchronous mock disbursement → status `disbursed` + `debit_withdraw` ledger entry, Σ ledger = balance (130000−5200−50000=74800); reject path restores/leaves balance untouched, no payment-service call. AC-6.c — review on a completed order_item succeeds; duplicate review on same order_item → 409 CONFLICT; review on non-completed order → 422 VALIDATION_ERROR; product `rating_avg`/`rating_count` correctly recomputed. AC-6.d — stopped `notification-service`, paid callback still returned 200 and transitioned the order (notification failure doesn't block the transaction); the `email:send` Asynq task was found in Redis `state: retry` with the gRPC unavailable error, confirming automatic retry rather than silent drop. `make lint && make test` green. Migrations verified clean via a disposable throwaway Postgres container: up (version 27, 32 tables) → down looped 27× to empty (only `schema_migrations`, 0 rows) → up again (version 27, 32 tables). `docker compose up` brings all 7 containers (postgres, redis, minio, api, worker, payment-service, notification-service) healthy/running, `/health` returns 200, no errors in any service log.
- **FR-7.8 OpenAPI authored by hand from `route.go`, not generated from Fiber annotations**: `docs/openapi.yaml` covers all 101 routes, grouped by tag matching the handler package boundary (auth, user, cart, checkout, order, shipment, review, wishlist, supplier[-product/-order], inventory, payout, report, admin[-supplier/-category/-order/-payout/-banner/-report], media, notification, public, system, internal). All responses reference a shared `components.schemas.Envelope` (matches `pkg/response.Envelope`) via `components.responses.OK`/`Error`; public/no-auth routes marked `security: []` per operation.
- **FR-7.8 route-vs-spec parity proven by `scripts/check_openapi_routes.py`**, a static regex parser of `route.go` (tracks `Group()` prefix chains, no fiber runtime needed) diffed against the parsed YAML `paths`. Verified 101/101 both directions (no missing, no extra). Fixed one false-negative in the fiber-path→OpenAPI-path normalizer that also now strips Fiber's trailing-slash routes (`/user/notifications/` → `/user/notifications`) to satisfy redocly's `no-path-trailing-slash` rule.
- **`npx @redocly/cli lint` used for AC-7.d's "OpenAPI valid" clause**: initial draft had 1 error (trailing slash) + 204 warnings (missing `operationId` + missing 4XX response on all 101 operations, missing `info.license`). Fixed via a scripted YAML rewrite (deterministic `operationId` slug per method+path, shared `components.responses.Error` referenced as each operation's `4XX`, `info.license` added) rather than 101 hand-edits. Final: 0 errors, 1 warning (`no-server-example.com` — intentionally flags the documented `http://localhost:8080` dev server, left as-is since this is a local-dev quickstart doc, not a hosted spec).
- **FR-7.8 README scoped to the explicit "<10 min from clone" requirement**: quickstart uses `make docker-up && make seed-demo && make run` (+ `make run-worker` in a second terminal), a demo-accounts table (3 roles, shared password from `seeds/demo.go`), and a `make help`-derived common-tasks list. Cross-checked every command against the real `Makefile` targets, `docker-compose.yml` services, and `.env.example` vars rather than writing from memory.
- **Bug found & fixed during AC-7.b verification: `/metrics` was rate-limited**, same 60 req/min/IP limiter as every other route (only `/health` was exempted). A Prometheus scraper hitting the app repeatedly from one source IP would get 429'd in production. Fixed by adding `/metrics` to the limiter's `Next` skip predicate alongside `/health` (`internal/delivery/http/middleware/rate_limit.go`).
- **`make docker-*` targets fixed to pass `--env-file .env`**: Docker Compose's `${VAR}` interpolation in the compose YAML (e.g. `INTERNAL_SECRET: "${APP_PAYMENT_INTERNAL_SECRET}"`) resolves against a `.env` colocated with the *invoked compose file* (`deploy/`), not the `env_file:`-referenced path (`../.env`, container-runtime only) — those are two separate lookups. `deploy/.env` didn't exist, so payment-service's actual `INTERNAL_SECRET` env var was silently empty despite the root `.env` having the right value, even though `docker compose up` reported all containers healthy. Fixed by adding `--env-file .env` to every `docker compose` invocation in the `Makefile` rather than duplicating a `deploy/.env` file (single source of truth). Same class of bug as the Phase 5/6 `.env`/`.env.example` drift issues, but at the interpolation layer instead of the variable-presence layer.
- **AC-7.a–7.c verification evidence**: AC-7.a — `make seed-demo` run against the docker-compose Postgres (not the host-local one `.env`'s `APP_DB_DSN` points at by default — two separate databases when running the containerized stack; seeded via a throwaway `golang:1.26-alpine` container on the compose network to reach `postgres:5432` directly), then walked all 3 roles live: buyer login → catalog browse (20 products) → `GET /user/me`; supplier login → `GET /supplier/products` (10 own products); admin login (`admin@bangunin.local` / `change-me`, not the demo `Password123!` — admin creds are separate per `.env` `SEED_ADMIN_*`) → `GET /admin/suppliers` (6 suppliers). All calls returned `success:true` with expected payload shapes. CI only triggers on push to `main`/`develop` or `pull_request` events (not arbitrary feature-branch pushes) — PR #9 was opened to trigger it, and its first run failed lint: `golangci-lint-action@v6`'s `version: latest` resolved to a stale v1.64.8 binary built with go1.24, which can't parse `go.mod`'s `go 1.25.0` directive. Fixed by bumping to `golangci-lint-action@v7` (v2-native, matches `.golangci.yml`'s `version: "2"`) pinned to `v2.12.2`; CI green on PR #9 run [30717625771](https://github.com/erdinhrmwn/bangunin-project/actions/runs/30717625771). AC-7.b — after the `/metrics` rate-limit fix above, `hey -z 60s -q 0.9 -c 1` against `GET /api/v1/products` (rate kept under the 60/min/IP security limiter rather than weakening it for a benchmark): 54/54 requests `200`, p95 = 24.7ms (well under 300ms), 0 errors, 0 5xx. `/metrics` confirmed scrapeable (200, unthrottled) with `http_request_duration_seconds_bucket{path="/api/v1/products"}` populated matching the run (53/54 under 25ms). `deploy/grafana-dashboard.json`'s 3 panel queries (`http_requests_total`, `http_request_duration_seconds_bucket`, `asynq_queue_size`) verified to reference metric names that exist verbatim in the live `/metrics` output. AC-7.c — `make vuln` (govulncheck): 0 vulnerabilities affecting project code (1 informational transitive-dependency finding not called by any code path, non-blocking). Live `curl -X POST /internal/payments/callback` with no `X-Internal-Secret` header → `401 {"success":false,"message":"Invalid internal secret"}`; same request with the correct secret → `200`.

### Post-Phase-7 Hardening (branch `fix/hardening-body-rate-limit`)

`/code-review` (high effort) ran against `develop` HEAD (`366993f..c2e2f92`, all Phase 1–7 merged). 10 findings CONFIRMED after verification; all 10 fixed on this branch, not tied to a specific FR/AC (cross-cutting hardening, not new requirement work) — logged here for traceability instead of the phase checklists above.

- `eff4c95` fix(middleware): body-limit middleware compared the raw `Content-Length` header (attacker-controlled, can be forged/omitted) instead of the actual body length — switched to measuring `len(c.Body())`.
- `79b31ae` fix(middleware): rate limiter Redis keys had no prefix, colliding with unrelated keys in the same Redis DB — added a `ratelimit:` prefix.
- `6ad9cf1` fix(app): `/metrics` was registered before the `Recover` middleware, so a panic while serving it would crash the process instead of being recovered — reordered.
- `78ec19b` fix(handler): `parseAuditRange` had `from`/`to` swapped, silently inverting every admin audit-log date filter.
- `2949fed` fix(repository): wishlist listing didn't exclude suspended suppliers or banned products, letting users add/see wishlisted items no longer purchasable.
- `ff3423e` fix(observability): Prometheus queue-size exporter goroutine was never stopped on shutdown and polled queues sequentially instead of concurrently.
- `df30632` fix(wishlist): `Catalog.Detail`'s per-user `IsWishlisted` lookup bypassed the product-detail cache entirely (unconditional Postgres hit on every cached-detail request) — made `IsWishlisted` itself cache-aware with its own per-user Redis key (60s TTL, invalidated on add/remove), since the shared product cache entry can't hold a per-user flag without leaking one user's state to all viewers.
- `1998e4c` perf(report): supplier/admin sales-summary usecases ran 3–5 independent repo queries sequentially — parallelized via `errgroup` (matches the existing `checkout.go` `Preview` pattern); `OrderRepository.PlatformSummary` combined two sequential queries against the same table/date-range into one `GROUP BY status` query.
- `8b8b263` refactor(worker): `report:generate`/`report:generate-admin` Asynq handlers each inlined the same ~65-line upload→presign→notify→email tail — extracted into a shared `deliverReport` function (mirrors the `recordAndNotify` precedent in `adminsupplier.go`/`payout.go`).
- `make lint && make test` (full `golangci-lint run ./...` + `go test ./...`, not per-file approximations) both green after all 10 fixes: 0 lint issues, all packages `ok` (deploy stack — postgres/redis/minio — running locally for the tests that need it).

### Post-Phase-7 Hardening, round 2 (branch `fix/post-review-hardening-2`)

Second `/code-review` (high effort) audit, 10 dimensions × 11 roles via parallel Explore subagents. Findings fixed one commit each, listed here as they land.

- fix(checkout): `Confirm`'s `ShippingCost` was client-supplied (echoed back from `Preview` by the client, per its own doc comment) and flowed unvalidated into `grandTotal` → the exact Xendit invoice amount — a price-manipulation gap. Fixed by caching every valid per-supplier shipping cost (courier options + fleet rate) computed during `Preview` into Redis (`checkout_quote:<userID>:<addressID>`, 15min TTL, via the existing but previously-unwired `redis.KVStore`), and validating each `Confirm` choice's `ShippingCost` against that cache. **Decision**: reject on mismatch/expiry (`QUOTE_EXPIRED`/`SHIPPING_COST_MISMATCH`, both 422) rather than silently substituting the server value — simpler to reason about/test and gives the client explicit feedback.
- test(postgres): added testcontainers-go coverage for `CheckoutGroupRepository.Confirm`, `PaymentRepository`, and `StockReservationRepository.ApplyAdjustments` against real Postgres — money-critical repos previously only had mocked-repo usecase tests, never a real driver. Uncovered a real stock-oversell bug in the process (see next entry).
- fix(checkout): `Confirm`'s per-variant availability check combined a `FOR UPDATE` lock and a `stock_reservations` sum into one SQL statement. Under READ COMMITTED, a transaction blocked on the lock only gets a fresh snapshot once the *locked row* is updated by the unblocking transaction — but `product_variants.stock` isn't touched until payment, so the blocked transaction's subquery ran against its stale pre-block snapshot, missed the other transaction's just-committed reservation, and let both concurrent `Confirm` calls pass the check, overselling stock. Reproduced deterministically with a new concurrent testcontainer test (2 goroutines racing the same variant, stock exactly exhausted by one). **Decision**: split into a lock-only query and a separate sum query issued only after the lock is granted, so the sum gets a fresh snapshot — rather than switching isolation levels, which would affect the whole transaction's other reads/writes. Also mapped Postgres deadlock (SQLSTATE `40P01`) on the lock query to `errs.ErrConflict`, consistent with the existing insufficient-stock conflict path (previously fell through to a generic wrapped error).
