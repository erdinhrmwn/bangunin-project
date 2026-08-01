# CLAUDE.md — Building Materials Marketplace ("Bangunin" / [final name TBD])

This document is the project's primary context. Read and follow every decision here before writing any code. Do not change architectural decisions without my explicit confirmation.

## 1. Project Description

Vertical marketplace specialized in building materials & construction supplies for Indonesia. Suppliers (distributors/building material stores) sell to contractors, builders, and project owners. Unique characteristics: high AOV (Rp1–5M), heavy materials (shipping cost based on actual vs. volumetric weight), distinctive selling units (sack, rod/bar, m³, box), escrow payments, order split per supplier.

## 2. Tech Stack (FINAL — do not change)

- **Go 1.23+**, HTTP framework **Fiber v3** (pin version in go.mod; v3 is still RC — isolate all Fiber code in `internal/delivery/http`)
- **PostgreSQL 16** (pgx + sqlc), **Redis 7** (cache, session, rate limit, stock lock, Asynq backend)
- **Asynq** for background jobs + scheduler
- **gRPC** to 2 separate external services: payment-service (Xendit) & notification-service (Mailjet). Contracts in `proto/`
- **RajaOngkir (Komerce)** via HTTP client for shipping cost & tracking
- Object storage: **Cloudflare R2** (S3-compatible; dev uses MinIO)
- zerolog, Viper, golang-migrate, go-playground/validator, golang-jwt/jwt v5, testify + mockery + testcontainers-go, golangci-lint
- Docker + docker-compose for local dev (postgres, redis, minio, api, worker)

## 3. Architecture & Directory Structure

Clean Architecture, dependencies flow inward: `delivery → usecase → domain ← repository/infra`.

```
cmd/{api,worker,migrate}/main.go
config/
internal/
  domain/{entity, repository(interface), service(external interface), errs}
  usecase/{auth,user,supplier,admin,category,product,inventory,catalog,cart,checkout,order,shipping,review,wishlist,payout,notification,report,media}
  delivery/http/{handler, middleware, dto, route}
  repository/{postgres, redis}
  infra/{database, redis, grpcclient, rajaongkir, storage, queue}
  app/{server.go, container.go}   # manual DI
pkg/{jwt, response, pagination, validator, hash, logger, apperr}
proto/  migrations/  seeds/  docs/  deploy/  scripts/  test/
```

Rules:
- All repository & external service interfaces are defined in `internal/domain`, implementations in `repository/` or `infra/`.
- Handlers contain NO business logic — only bind DTO, call usecase, map error → response.
- User output files: `/mnt` is not relevant — this is a regular local repo.

## 4. Domain & Data Schema (summary — details in docs/erd.md)

Core tables (±28): roles, users (direct role_id FK — NO pivot table), user_addresses, suppliers (1:1 users, status pending/approved/rejected/suspended, origin_city_id, own_fleet_enabled), supplier_documents, supplier_bank_accounts, categories (hierarchical parent_id), products (specs jsonb, search_vector tsvector+GIN, denormalized rating), product_variants (price, weight_gram REQUIRED, dimensions, unit, min_order_qty, stock), product_images, stock_movements (append-only), stock_reservations (active/converted/released, expires_at), carts, cart_items, checkout_groups, payments (1:1 checkout_group, xendit_invoice_id), orders (split per supplier under checkout_group, snapshot shipping_address jsonb, commission_amount), order_items (snapshot name/price/weight), order_status_histories, shipments (method courier|supplier_fleet), reviews (1:1 order_item), supplier_balances, ledger_entries (append-only, balance_after), withdraw_requests, notifications, wishlists, audit_logs, banners.

ID: UUID v7. Timestamps: timestamptz. Soft delete only when needed (default: no).

## 5. Critical Design Decisions (MUST be followed)

1. **Order split per supplier**: 1 checkout → N orders (per supplier) under 1 checkout_group → 1 Xendit invoice. Shipping cost calculated per order.
2. **Shipping cost**: chargeable weight = max(actual weight, volumetric (l×w×h/6000)). Support 2 methods: RajaOngkir courier & supplier fleet (flat/km zone).
3. **Stock reservation**: checkout confirm → Redis lock per variant → validate → insert reservation (expires in 2 hours). Paid → convert to stock_movements + decrement stock. Expired → release job. NEVER deduct stock directly at checkout.
4. **Escrow + ledger**: funds enter the platform. Order completed → ledger credit_order + debit_commission (4% commission) → supplier balance. Withdraw → admin approval → disbursement via payment-service. `ledger_entries` is append-only, no UPDATE/DELETE allowed.
5. **Order state machine**: pending_payment → paid → processed → shipped → delivered → completed; branches expired/cancelled(+refund). Transitions validated in usecase per actor (supplier/user/admin/system), all recorded in order_status_histories.
6. **RBAC**: `Auth()` + `RequireRole(role)` middleware; route group `/api/v1/{public|user|supplier|admin}`.
7. **Standard response envelope** (pkg/response): `{success, message, data, meta, errors}`. Domain errors mapped to HTTP status via pkg/apperr.
8. **Snapshot** price/name/address in order_items & orders — do not join to master data for historical order data.

## 6. Background Jobs (Asynq)

`email:send`, `order:expire` (cancel + release reservation), `order:autocomplete` (N days after delivered), `payment:sync`, `stock:release`, `media:process` (image resize), `report:generate`, `notification:lowstock` (cron).

## 7. Code Conventions

- Code/comment/commit language: English. User-facing error messages (response message): English.
- File names snake_case, short singular package names.
- Every usecase has a unit test (mock repo via mockery). Repository tested with testcontainers.
- Migrations: one change = one up/down file pair. Do not edit an already-created migration — create a new one.
- DTO validation in the delivery layer (validator tags), business validation in usecase.
- Never commit secrets. All config via env (see .env.example).
- After significant changes: `make lint && make test` must be green before proceeding.

## 8. Phase Roadmap (work sequentially, one phase = one series of PRs)

1. **Foundation**: skeleton per structure above, config, PG+Redis connections, logger, response envelope, apperr, basic middleware (recover, requestid, logger, ratelimit), initial migrations (roles, users), seeds roles+admin, docker-compose, Makefile (run, migrate, seed, test, lint, gen-proto), health endpoint.
2. **Auth + RBAC**: register/login/refresh/logout, email verification OTP (email:send job → stub notification-service for now), password reset, Auth+RequireRole middleware.
3. **Supplier onboarding**: store registration, document upload (media module + MinIO), admin approval/reject + audit_logs.
4. **Catalog**: category CRUD (admin), product+variant+image CRUD (supplier), inventory & movements, public catalog (tsvector search, filter, cursor pagination).
5. **Core transactions**: cart → checkout preview (RajaOngkir + fleet shipping cost) → confirm (lock+reservation+order split) → payment-service gRPC (implement the actual service in a separate repo/services/ folder) → paid webhook → order state machine → shipment.
6. **Post-transaction**: review, in-app notifications, payout (ledger, withdraw, approval, disbursement).
7. **Complementary**: wishlist, report, banner, hardening + end-to-end integration test.

## 9. How I Expect Claude Code to Work

- Before starting a phase: show the plan of files to be created/changed, wait for my confirmation for major phases.
- Work in chunks that can be compiled & tested — don't write 50 files at once without ever running a build.
- If there's ambiguity against this document: ask first; don't invent new architectural decisions.
- Update PROGRESS.md every time a sub-task is completed (checklist per phase).

## 10. Git Workflow

### Initial Setup

1. Initialize a git repo if not already done. Main branch: `main` (release only). Integration branch: `develop` (all phase PRs target this). `main` is only updated by merging `develop` at release points.
2. First commit: all planning documents (CLAUDE.md, docs/) with message `docs: initial project planning documents`.
3. Create PROGRESS.md containing a checklist of ALL FR and AC from Phase 1–7 (copied from docs/requirements.md), all unchecked. Commit: `docs: add progress tracker`.
4. If the GitHub remote isn't set up yet, ask for the repo URL. Verify `gh auth status` works — if not, report it and use the fallback.

### Per-Phase Cycle

For each phase N, run this sequence:

- **A. BRANCH** — from the latest `develop`, create branch: feat/phase-N-<short-name> (e.g., feat/phase-1-foundation, feat/phase-2-auth-rbac). Direct commits to `main` or `develop` are FORBIDDEN. One branch = one phase, don't mix.
- **B. PLAN** — show the implementation plan: list of files to be created/changed, mapped to each FR-N.x, plus the work order as a list of small sub-tasks. WAIT for approval before writing code.
- **C. IMPLEMENTATION** — work through sub-tasks one by one:
  - Every sub-task must end in a buildable state (run the relevant build + test before committing).
  - Commit EVERY completed sub-task, however small. Don't pile up multiple pieces of work into one commit.
  - Commit format: conventional commits + requirement reference. Example: `feat(auth): implement login with brute force guard [FR-2.3]`, `test(checkout): add concurrent stock race test [AC-5.b]`, `fix(order): reject illegal status transition [FR-5.8]`.
  - Every completed FR: check it off in PROGRESS.md and include it in the same commit ("... + update progress").
  - If ambiguity is encountered: stop, ask, record the decision in PROGRESS.md's "Decisions" section.
- **D. PHASE VERIFICATION** — run all AC-N.x. The phase is only done when: all ACs pass, `make lint && make test` is green, migrations up/down are clean from an empty database, and `docker compose up` is healthy. Report the status of each AC one by one (pass/fail + brief evidence).
- **E. PULL REQUEST** — once all ACs pass:
  - Check off all FR & AC for the phase in PROGRESS.md, final commit: `docs: complete phase N progress checklist`.
  - Push the branch, then create a PR to `develop` (not `main`) via `gh pr create` with: title in conventional commits format (e.g. `feat: implement phase N <phase name>`), body containing a summary of changes, the FR & AC checklist that passed, manual testing steps, and notes on decisions made.
  - Fallback if gh/remote isn't available: just push the branch (or commit locally), then write a draft PR description to `docs/pr/phase-N.md` and report that a manual PR needs to be created.
- **F. STOP & WAIT** — don't start the next phase before confirmation that the PR has been reviewed/merged into `develop`. Once confirmed to proceed: checkout `develop`, pull, repeat the cycle from step A for phase N+1.

### Additional Rules

- Never force push, never rebase/rewrite history that's already been pushed, never delete a branch before its PR is merged.
- Never commit secret/.env files — make sure .gitignore is correct from Phase 1 onward.
- If a session is interrupted: read CLAUDE.md + PROGRESS.md + `git log --oneline -20` and `git status` to recover context, continue from the unchecked sub-task on the currently active phase branch.
