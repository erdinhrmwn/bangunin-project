# CLAUDE.md — Building Materials Marketplace ("Bangunin" / [final name TBD])

This document is the project's primary context. Read and follow every decision here before writing any code. Do not change architectural decisions without my explicit confirmation.

## 1. Project Description

Vertical marketplace specialized in building materials & construction supplies for Indonesia. Suppliers (distributors/hardware stores) sell to contractors, builders, and project owners. Unique characteristics: high AOV (Rp1–5M), heavy materials (shipping cost based on actual weight vs volumetric), material-specific units (sack, rod/bar, m³, box), escrow payment, order split per supplier.

## 2. Tech Stack (FINAL — do not change)

- **Go 1.25+** (Fiber v3 minimum requirement), HTTP framework **Fiber v3** (STABLE RELEASE v3.0+; handler `func(c fiber.Ctx) error` without pointer, binding via `c.Bind().Body()` — see docs/packages-reference.md §1; isolate all Fiber code in `internal/delivery/http`)
- **PostgreSQL 16** (pgx + sqlc), **Redis 7** (cache, session, rate limit, stock lock, Asynq backend)
- **Asynq** for background jobs + scheduler
- **Payment (Xendit), Notification (Mailjet) & Shipping (RajaOngkir/Komerce)**: internal adapter packages implementing the `domain/service` interfaces (`PaymentService`, `NotificationService`, `ShippingService` in `internal/infra/xendit`, `internal/infra/mailjet`, `internal/infra/rajaongkir`) — plain HTTP clients called in-process, no separate service from day one. The interface is the swap point: changing vendor later means writing a new `internal/infra/<vendor>` implementation, not touching the usecase. Extract any of them to a standalone gRPC service later only if a real need appears (see §5.9).
- Object storage: **Cloudflare R2** (S3-compatible; use MinIO for dev)
- zerolog, Viper, golang-migrate, go-playground/validator, golang-jwt/jwt v5, testify + mockery + testcontainers-go, golangci-lint
- Docker + docker-compose for local dev (postgres, redis, minio, api, worker)

## 3. Architecture & Directory Structure

Clean Architecture, dependencies flow inward: `delivery → usecase → domain ← repository/infra`.

```
cmd/{api,worker,migrate}/main.go
config/
internal/
  domain/{entity, repository(interface), service(external interface: PaymentService, NotificationService, ShippingService, StorageService), errs}
  usecase/            # FLAT — single `usecase` package, 1 file per module: auth_usecase.go, user_usecase.go, supplier_usecase.go, admin_usecase.go, category_usecase.go, product_usecase.go, inventory_usecase.go, catalog_usecase.go, cart_usecase.go, checkout_usecase.go, order_usecase.go, shipping_usecase.go, review_usecase.go, wishlist_usecase.go, payout_usecase.go, notification_usecase.go, report_usecase.go, media_usecase.go (test: *_usecase_test.go alongside). DO NOT create a subfolder per module inside usecase.
  delivery/http/{handler, middleware, dto, route}
  repository/{postgres, redis}
  infra/{database, cache, xendit, mailjet, rajaongkir, storage, queue}   # `cache` = Redis client init (not `redis`, to avoid name collision with repository/redis); `xendit`/`mailjet`/`rajaongkir` = current vendor implementations of the domain/service interfaces, no separate services
  app/{server.go, container.go}   # manual DI
pkg/{jwt, response, pagination, validator, hash, logger, apperr}
migrations/  seeds/  docs/  deploy/  scripts/  test/
```

Rules:
- All repo interfaces & external service interfaces are defined in `internal/domain`, implementations in `repository/` or `infra/`.
- Handlers contain NO business logic — only bind DTO, call usecase, map error → response.
- User output files: `/mnt` is not relevant here — this is a regular local repo.

## 4. Domain & Data Schema (summary — details in docs/erd.md)

Core tables (~28): roles, users (role_id direct FK — NO pivot table), user_addresses, suppliers (1:1 users, status pending/approved/rejected/suspended, origin_city_id, own_fleet_enabled), supplier_documents, supplier_bank_accounts, categories (hierarchical parent_id), products (specs jsonb, search_vector tsvector+GIN, rating denormalized), product_variants (price, weight_gram REQUIRED, dimensions, unit, min_order_qty, stock), product_images, stock_movements (append-only), stock_reservations (active/converted/released, expires_at), carts, cart_items, checkout_groups, payments (1:1 checkout_group, xendit_invoice_id), orders (split per supplier under checkout_group, snapshot shipping_address jsonb, commission_amount), order_items (snapshot name/price/weight), order_status_histories, shipments (method courier|supplier_fleet), reviews (1:1 order_item), supplier_balances, ledger_entries (append-only, balance_after), withdraw_requests, notifications, wishlists, audit_logs, banners.

ID: UUID v7. Timestamps: timestamptz. Soft delete only if needed (default: no).

## 5. Critical Design Decisions (MUST be followed)

1. **Split order per supplier**: 1 checkout → N orders (per supplier) under 1 checkout_group → 1 Xendit invoice. Shipping cost calculated per order.
2. **Shipping cost**: chargeable weight = max(actual weight, volumetric (w×l×h/6000)). Support 2 methods: RajaOngkir courier & supplier fleet (flat/km zone).
3. **Stock reservation**: checkout confirm → Redis lock per variant → validation → insert reservation (expires in 2 hours). Paid → convert to stock_movements + decrement stock. Expired → release job. NEVER decrement stock directly at checkout.
4. **Escrow + ledger**: funds go to the platform. Order completed → ledger credit_order + debit_commission (4% commission) → supplier balance. Withdraw → admin approval → disbursement via `PaymentService` (Xendit Disbursement API). `ledger_entries` is append-only, no UPDATE/DELETE allowed.
5. **Order state machine**: pending_payment → paid → processed → shipped → delivered → completed; branches expired/cancelled(+refund). Transitions validated in usecase per actor (supplier/user/admin/system), all logged in order_status_histories.
6. **RBAC**: middleware `Auth()` + `RequireRole(role)`; route group `/api/v1/{public|user|supplier|admin}`.
7. **Standard response envelope** (pkg/response): `{success, message, data, meta, errors}`. Domain errors mapped to HTTP status via pkg/apperr.
8. **Snapshot** price/name/address in order_items & orders — do not join to master data for historical order data.
9. **Payment/Notification/Shipping as internal adapters, not microservices**: `internal/infra/xendit`, `internal/infra/mailjet`, and `internal/infra/rajaongkir` implement the `domain/service.PaymentService` / `NotificationService` / `ShippingService` interfaces directly in-process — no separate proto contracts, no extra containers, no cross-service webhook hop. Xendit/Mailjet webhooks land directly on a monolith HTTP endpoint. Extract any of them to a standalone gRPC service later only when a real need shows up (separate team/deploy cadence, independent scaling) — the interface boundary already makes that a swap, not a rewrite.

## 6. Background Jobs (Asynq)

`email:send`, `order:expire` (cancel + release reservation), `order:autocomplete` (N days after delivered), `payment:sync`, `stock:release`, `media:process` (image resize), `report:generate`, `notification:lowstock` (cron).

## 7. Code Conventions

- **BEFORE using a package's API, read docs/packages-reference.md first** — it's the primary source of truth for syntax & patterns of every dependency (Fiber v3, pgx, sqlc, Asynq, etc.). Don't guess an API from memory and don't use Fiber v2 patterns. If you find an API that differs from the document, fix the document in the same commit.
- Language for code/comments/commits: English. Error messages to users (response message): Indonesian.
- File names snake_case, short singular package names.
- Every usecase has a unit test (mock repo via mockery). Repositories are tested with testcontainers.
- Migrations: one change = one up/down file pair. Do not edit migrations that already exist — create a new migration.
- DTO validation in the delivery layer (validator tags), business validation in the usecase.
- Never commit secrets. All configuration via env (see .env.example).
- After significant changes: `make lint && make test` must be green before proceeding.

## 8. Phase Roadmap (work sequentially, one phase = one series of PRs)

1. **Foundation**: skeleton matching the structure above, config, PG+Redis connections, logger, response envelope, apperr, base middleware (recover, requestid, logger, ratelimit), initial migrations (roles, users), seeds roles+admin, docker-compose, Makefile (run, migrate, seed, test, lint), health endpoint.
2. **Auth + RBAC**: register/login/refresh/logout, email OTP verification (email:send job → stub `NotificationService` implementation for now), password reset, Auth+RequireRole middleware.
3. **Supplier onboarding**: store registration, document upload (media module + MinIO), admin approval/reject + audit_logs.
4. **Catalog**: category CRUD (admin), product+variant+image CRUD (supplier), inventory & movements, public catalog (tsvector search, filter, cursor pagination).
5. **Core transactions**: cart → checkout preview (`ShippingService` via `internal/infra/rajaongkir` + fleet) → confirm (lock+reservation+split order) → `PaymentService` via `internal/infra/xendit` (real Xendit Invoice API calls) → paid webhook → order state machine → shipment.
6. **Post-transaction**: review, in-app notifications, payout (ledger, withdraw, approval, disbursement).
7. **Complementary**: wishlist, report, banner, hardening + end-to-end integration tests.

## 9. Expected Way of Working from Claude Code

- Before starting a phase: show the plan of files to be created/changed, wait for my confirmation for major phases.
- Work in chunks that compile & can be tested — don't write 50 files at once without ever running a build.
- If there's ambiguity against this document, ask first; don't invent new architectural decisions.
- Update PROGRESS.md every time a sub-task is completed (checklist per phase).

## 10. Git Workflow

### Initial Setup

1. Initialize git repo if not already done. Main branch: main.
2. First commit: all planning documents (CLAUDE.md, docs/) with message docs: initial project planning documents.
3. Create PROGRESS.md containing a checklist of ALL FRs and ACs from Phases 1–7 (copy from docs/requirements.md), all unchecked. Commit: docs: add progress tracker.
4. If GitHub remote is not set up, ask for the repo URL. Verify gh auth status works — if not, report it and use a fallback.

### Per-Phase Cycle

For each phase N, run this sequence:

- **A. BRANCH** — from latest main, create branch: feat/fase-N-<short-name> (example: feat/fase-1-fondasi, feat/fase-2-auth-rbac). Committing directly to main is FORBIDDEN. One branch = one phase, don't mix.
- **B. PLAN** — show the implementation plan: list of files to be created/changed, mapped to each FR-N.x, plus the work order as a list of small sub-tasks. WAIT for approval before writing code.
- **C. IMPLEMENTATION** — work through sub-tasks one by one:
  - Every sub-task must end in a buildable state (run build + related tests before commit).
  - Commit EVERY completed sub-task, no matter how small. Don't stack a lot of work into one commit.
  - Commit format: conventional commits + requirement reference. Example: feat(auth): implement login with brute force guard [FR-2.3], test(checkout): add concurrent stock race test [AC-5.b], fix(order): reject illegal status transition [FR-5.8].
  - When an FR is done: check it off in PROGRESS.md and include it in the same commit ("... + update progress").
  - If you encounter ambiguity: stop, ask, record the decision in the PROGRESS.md "Decisions" section.
- **D. PHASE VERIFICATION** — run all AC-N.x. Phase is done only if: all ACs pass, make lint && make test is green, migration up/down is clean from an empty database, and docker compose up is healthy. Report the status of each AC one by one (pass/fail + brief evidence).
- **E. PULL REQUEST** — after all ACs pass:
  - Check off all phase FRs & ACs in PROGRESS.md, final commit: docs: complete fase N progress checklist.
  - Push branch, then create a PR to main via gh pr create with: title "Fase N: <phase name>", body containing a summary of changes, checklist of passed FRs & ACs, manual testing instructions, and notes on decisions made.
  - Fallback if gh/remote is unavailable: just push the branch (or commit locally), then write a draft PR description to docs/pr/fase-N.md and report that a manual PR needs to be created.
- **F. STOP & WAIT** — don't start the next phase before confirmation that the PR has been reviewed/merged. Once confirmed to proceed: checkout main, pull, repeat the cycle from step A for phase N+1.

### Additional Rules

- Never force push, never rebase/alter already-pushed history, never delete a branch before the PR is merged.
- Don't commit secret/.env files — make sure .gitignore is correct from Phase 1.
- If a session is interrupted: read CLAUDE.md + PROGRESS.md + git log --oneline -20 and git status to recover context, continue from the unchecked sub-task on the currently active phase branch.
