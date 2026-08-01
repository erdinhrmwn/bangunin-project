# Requirements per Phase — Building Materials Marketplace

Implementation reference document. Each phase contains: goal, scope, functional requirements (FR), endpoints, migrations, background jobs, and acceptance criteria (AC) that must pass before the phase is considered done. Stored as `docs/requirements.md` in the repo — CLAUDE.md refers to this document.

Convention: `FR-x.y` = functional requirement, phase x number y. All endpoints under prefix `/api/v1`. All responses use the standard envelope `{success, message, data, meta, errors}`.

---

## PHASE 1 — Foundation

**Goal:** repo can be cloned, `docker compose up`, migrate, seed, and serve the health endpoint with the full backbone (config, DB, Redis, logger, middleware, error handling) ready for the next phases.

**In scope:** skeleton, tooling, local infrastructure, initial migrations, basic middleware.
**Out of scope:** all business features, gRPC client, RajaOngkir, storage.

### Functional Requirements
- **FR-1.1 Project structure** per CLAUDE.md §3. Every folder contains at minimum a `doc.go` or a real file — no unused empty folders.
- **FR-1.2 Config**: Viper reads `config.yaml` + env override (prefix `APP_`). Typed config struct: App{Name, Env, Port}, DB{DSN, MaxConns...}, Redis, JWT{Secret, AccessTTL, RefreshTTL}, and empty sections for Xendit/RajaOngkir/R2 (filled in during the relevant phase). App fails fast with a clear message if required config is missing.
- **FR-1.3 Database**: pgxpool connection with ping at startup, graceful close on shutdown. Migrations via golang-migrate run through `cmd/migrate` and a Makefile target.
- **FR-1.4 Initial migrations**: `roles` (seed: admin, supplier, user) and `users` per ERD (uuid v7 default, role_id FK, unique email, index on email & status). Both up and down work.
- **FR-1.5 Seeder**: idempotent (safe to run repeatedly). Seeds roles + 1 default admin account from env (`SEED_ADMIN_EMAIL`, `SEED_ADMIN_PASSWORD`).
- **FR-1.6 Redis**: client with ping at startup; get/set/del helper with TTL in `internal/repository/redis`.
- **FR-1.7 Logger**: zerolog JSON to stdout; level from config; standard fields (service, env). Request log middleware records method, path, status, latency, request_id.
- **FR-1.8 Basic middleware**: recover (panic → 500 envelope + log stack), request ID (header `X-Request-ID`, generated if missing), configurable CORS, Redis sliding-window rate limit (default 60 req/min/IP, path-exempt: /health).
- **FR-1.9 Response & error**: `pkg/response` (Success, Error, paginated meta) and `pkg/apperr` — AppError{Code, Message, HTTPStatus} type + mapper from domain errs (`ErrNotFound`→404, `ErrValidation`→422, `ErrUnauthorized`→401, `ErrForbidden`→403, `ErrConflict`→409, default 500 without leaking internal detail).
- **FR-1.10 Health**: `GET /health` → app status, db (ping), redis (ping), version, uptime. 503 if a dependency is down.
- **FR-1.11 Worker skeleton**: `cmd/worker` runs an Asynq server connected to Redis with 1 dummy task `system:heartbeat` registered; scheduler ready but empty.
- **FR-1.12 Tooling**: Makefile (run, run-worker, build, migrate-up/down/create, seed, test, lint, gen-proto placeholder), `.golangci.yml`, full `.env.example`, docker-compose (postgres 16, redis 7, minio, api, worker) with healthcheck and persistent volume.
- **FR-1.13 Graceful shutdown**: SIGINT/SIGTERM → stop accepting requests, wait for in-flight up to 10s, close DB/Redis.

### Acceptance Criteria
- AC-1.a `docker compose up -d` → all containers healthy; `make migrate-up && make seed` succeeds; `curl /health` = 200 with db & redis status "ok".
- AC-1.b Stop Redis → /health = 503; a synthetic panic in a test handler → clean 500 envelope + stack in log, process doesn't die.
- AC-1.c `make lint` and `make test` green; migrate down then up succeeds again; seed run 2x with no error/duplicate.
- AC-1.d Send 70 req/min to a test endpoint → some are rejected with 429 using the standard envelope.

---

## PHASE 2 — Auth + RBAC

**Goal:** full account lifecycle: register, email verification, login/refresh/logout, password reset, and per-role route protection.

**In scope:** user & supplier-account auth, JWT, email OTP (via notification stub), RBAC middleware.
**Out of scope:** supplier/store profile (phase 3), OAuth/social login.

### Functional Requirements
- **FR-2.1 Register** `POST /auth/register` {name, email, password, phone, role: user|supplier}. Password min 8 chars with a digit; email unique (409 if it already exists); the admin role CANNOT be registered through this endpoint. Password hashed with bcrypt cost 12. Account status `active`, `email_verified_at` null.
- **FR-2.2 Email verification**: after register, generate a 6-digit OTP, store in Redis `otp:verify:{userID}` TTL 10 min, enqueue `email:send`. `POST /auth/verify-email` {email, otp} → sets email_verified_at. `POST /auth/resend-otp` rate-limited 1x/min per email. Wrong OTP 5x → OTP invalidated.
- **FR-2.3 Login** `POST /auth/login` {email, password} → access token (JWT, TTL 15 min, claims: sub, role, jti) + refresh token (random opaque, TTL 7 days, stored in Redis `refresh:{token}`→userID). Unverified email → 403 with dedicated code `EMAIL_NOT_VERIFIED`. Suspended/banned account → 403. Brute force guard: 10 failures/15 min per email+IP → temporary block.
- **FR-2.4 Refresh** `POST /auth/refresh` {refresh_token} → rotation: old token deleted, new pair issued. Unknown token → 401.
- **FR-2.5 Logout** `POST /auth/logout` (auth) → deletes refresh token + blacklists access token jti in Redis until expiry.
- **FR-2.6 Password reset**: `POST /auth/forgot-password` (always 200 so email existence isn't leaked; sends OTP if it exists) → `POST /auth/reset-password` {email, otp, new_password} → invalidates all of the user's refresh tokens.
- **FR-2.7 Auth middleware**: parses Bearer JWT, rejects expired/blacklisted/invalid (401), injects claims into context (`ctxutil.UserID(c)`, `ctxutil.Role(c)`).
- **FR-2.8 RequireRole(roles...) middleware**: 403 if the role doesn't match. Route groups `/user`, `/supplier`, `/admin` wired up and proven via a `GET {group}/me` endpoint.
- **FR-2.9 Basic profile**: `GET /user/me`, `PATCH /user/me` (name, phone), `PATCH /user/me/password` (verifies old password).
- **FR-2.10 Notification stub**: log-only implementation of `domain/service.NotificationService` (prints email content to the log) in `infra/grpcclient` — used until the real service is available; swap the implementation without changing the usecase.

### Acceptance Criteria
- AC-2.a Happy path register→OTP (read from stub log)→verify→login→access `/user/me`→refresh→logout→old access token rejected 401.
- AC-2.b A `user`-role user accessing `/supplier/me` → 403; no token → 401; expired token → 401.
- AC-2.c A refresh token that's already been rotated, reused → 401. Password reset invalidates all old sessions.
- AC-2.d Auth usecase unit test coverage for: duplicate email, wrong/expired/5x-failed OTP, brute force lock, admin role rejected on register.

---

## PHASE 3 — Supplier Onboarding

**Goal:** supplier accounts complete a store profile, upload documents, and get approved/rejected by an admin with an audit trail.

**In scope:** store profile, KYC documents, bank account, media upload (MinIO/R2), admin approval, audit log, status notifications.
**Out of scope:** products (phase 4).

### Functional Requirements
- **FR-3.1 Media module**: `POST /media/upload` (auth; multipart) → validates type (jpg/png/webp/pdf) & size (images ≤5MB, pdf ≤10MB) → stores to bucket with key `{scope}/{uuid}.{ext}` → returns url + key. Enqueues `media:process` to resize images (max 1600px) — the worker replaces the original file. Implementation of `domain/service.StorageService` backed by the S3 API (MinIO locally, R2 in production).
- **FR-3.2 Store profile**: `POST /supplier/profile` & `PUT /supplier/profile` {store_name, description, origin_city_id, pickup_address, own_fleet_enabled, fleet_coverage_km?, fleet_flat_rate?}. Unique slug generated from store_name. origin_city_id required to be valid (for this phase an integer > 0 is enough; RajaOngkir validation follows in phase 5). One user can have at most one supplier (409).
- **FR-3.3 Documents**: `POST /supplier/documents` {doc_type: nib|ktp|npwp, file_key} — max 1 active document per type (re-uploading replaces it, status resets to pending). `GET /supplier/documents`.
- **FR-3.4 Bank account**: CRUD `/supplier/bank-accounts`; exactly one is_default; account number 6–20 digits.
- **FR-3.5 Submit application**: `POST /supplier/submit` — only allowed if the profile is complete + at minimum KTP & NIB documents exist. Supplier status: `draft → pending`. Cannot submit while already pending/approved.
- **FR-3.6 Admin review**: `GET /admin/suppliers?status=&q=&page=` (list + filter), `GET /admin/suppliers/:id` (detail + documents + presigned url), `POST /admin/suppliers/:id/approve` (sets approved + verified_at), `POST /admin/suppliers/:id/reject` {reason} (reason required; supplier can fix and resubmit), `POST /admin/suppliers/:id/suspend` {reason}.
- **FR-3.7 Audit log**: every approve/reject/suspend recorded in audit_logs {actor_id, action, entity, metadata: {reason, before, after}, ip}. `GET /admin/audit-logs` with action & date-range filters.
- **FR-3.8 Notifications**: supplier status change → in-app notification (notifications table) + enqueue email. `GET /user/notifications` + `POST /user/notifications/:id/read` + unread count.
- **FR-3.9 Guard**: all `/supplier/*` selling-related endpoints in later phases must pass through the `RequireApprovedSupplier` middleware (403 `SUPPLIER_NOT_APPROVED` if not approved) — this middleware is built in this phase.

### Acceptance Criteria
- AC-3.a Full flow: supplier completes profile → uploads 2 documents → submits → admin rejects with a reason → supplier receives notification + reason → fixes it → resubmits → approved → notification received, verified_at set, audit_logs has 2 entries.
- AC-3.b Uploading a .exe or a 20MB file → 422. Large images get resized by the worker (verify output dimensions).
- AC-3.c Submit without NIB → 422 with a list of what's missing. Accessing a (dummy) selling endpoint before approval → 403.
- AC-3.d New migrations: suppliers, supplier_documents, supplier_bank_accounts, notifications, audit_logs — up/down clean.

---

## PHASE 4 — Catalog

**Goal:** approved suppliers manage a full product catalog; the public can search and browse quickly.

**In scope:** category, product+variant+image, stock & movements, public listing + full-text search, cache.
**Out of scope:** cart/checkout (phase 5), review (phase 6).

### Functional Requirements
- **FR-4.1 Category (admin)**: hierarchical CRUD `/admin/categories` (parent_id, max 3 levels), unique slug, is_active, sort_order. Delete only if it has no children & no products (409). Public: `GET /categories` (tree, Redis cache 10 min, invalidated on mutation). Seed ~20 realistic building-material categories (Cement & Aggregate, Steel & Iron, Wood, Paint, Tile & Granite, Sanitary, Roofing, Electrical, Pipe, etc.).
- **FR-4.2 Product (supplier, approved-only)**: CRUD `/supplier/products`. Fields: category_id, name, description, specs (free-form jsonb: brand, material, dimensions), status draft|active|inactive. Publish (draft→active) only if ≥1 active variant and ≥1 image. Globally unique slug.
- **FR-4.3 Variant**: nested CRUD `/supplier/products/:id/variants`. Required fields: name, sku (unique per supplier), unit (enum: sak|batang|m3|dus|pcs|lembar|kg|roll|set), price>0, **weight_gram>0**, dimensions in cm (may be 0 for non-volumetric), min_order_qty≥1, stock≥0. Price updates don't change existing orders (snapshot — enforced in phase 5).
- **FR-4.4 Images**: attached from media upload, max 8 per product, exactly one is_primary, sortable.
- **FR-4.5 Stock**: `POST /supplier/variants/:id/stock-adjustment` {qty(+/-), note} → writes stock_movements (type in/out/adjustment, stock_after) in a transaction together with the stock update; stock can't go negative (409). `GET /supplier/variants/:id/movements` (paginated history).
- **FR-4.6 Public catalog**: `GET /products?q=&category=&min_price=&max_price=&city=&sort=(newest|price_asc|price_desc|rating)&cursor=` — only active products from approved suppliers. q uses tsvector (name+description+specs brand) websearch_to_tsquery, ranked with ts_rank. Compact response: id, slug, name, primary_image, price_min–max, unit, supplier{name, city}, rating. Cursor-based pagination.
- **FR-4.7 Public detail**: `GET /products/:slug` → full active variants + available stock (stock − active reservations; 0 before phase 5), images, specs, brief supplier profile. Redis cache 60s, invalidated when the supplier edits the product.
- **FR-4.8 search_vector** kept in sync by a PostgreSQL trigger (INSERT/UPDATE), GIN index. EXPLAIN uses the index.
- **FR-4.9 Public store**: `GET /suppliers/:slug` → profile + its products (paginated).
- **FR-4.10 Low-stock job**: daily `notification:lowstock` scheduler — variant stock < threshold (default 5, per-supplier configurable later; hardcoded default for now) → notify supplier, max 1x/day/variant.

### Acceptance Criteria
- AC-4.a Supplier creates product "Portland Cement 40kg" with 2 variants + 3 images → publishes → shows up in `GET /products?q=semen` and searching "portland 40" also finds it.
- AC-4.b Publish without an image → 422. Variant without weight_gram → 422. Duplicate SKU within one supplier → 409.
- AC-4.c Adjustment of −10 on stock of 5 → 409; +10 then −3 → 2 movement rows with correct stock_after.
- AC-4.d Suspended supplier's products don't show up in the public catalog. Cache: renaming a product → public detail updates ≤ invalidation window (not waiting for the 60s TTL).
- AC-4.e Repository search integration test with testcontainers: seed 50 products, query q+filter+sort returns correct results and ordering.

---

## PHASE 5 — Core Transactions

**Goal:** end-to-end money and goods flow: cart → checkout (shipping) → pay via payment-service (Xendit) → order runs through a state machine → shipped → received.

**In scope:** cart, checkout preview/confirm, RajaOngkir client, stock reservation + Redis lock, payment-service gRPC (built for real), webhook, order lifecycle, shipment, expire job.
**Out of scope:** review, payout (phase 6).

### Functional Requirements
- **FR-5.1 Cart**: `GET /user/cart`, `POST /user/cart/items` {variant_id, qty}, `PATCH .../items/:id`, `DELETE .../items/:id`. Qty merges for the same variant. Response grouped per supplier, each item carries issue flags: out_of_stock, below_min_order, price_changed (since added), product_inactive. A problem item doesn't block viewing the cart but does block checkout.
- **FR-5.2 RajaOngkir client** (`infra/rajaongkir`): implementation of `domain/service.ShippingGateway` — GetCost(originCityID, destCityID, weightGram, couriers[]) and (stub for this phase) TrackWaybill. 5s timeout, 1 retry, error → `ErrShippingUnavailable`. API key configured via env; provide a mock mode (`RAJAONGKIR_MOCK=true`) returning deterministic rates for dev/test.
- **FR-5.3 Checkout preview** `POST /user/checkout/preview` {address_id, items[]}: validate every item (stock, min order, active, supplier approved) → group per supplier → per group compute chargeable weight = Σ max(weight, volumetric l·w·h/6000) → call shipping in parallel per group → return: groups[{supplier, items, subtotal, shipping_options[courier/service/etd/cost + supplier-fleet option if enabled & distance within coverage]}], grand_total_estimate. Address must belong to the user.
- **FR-5.4 Checkout confirm** `POST /user/checkout/confirm` {address_id, items[], shipping_choice per group}: atomic sequence — (1) Redis lock `stock:{variantID}` (SETNX+TTL 10s, sorted by ID to avoid deadlock), (2) DB TX: re-validate price & effective stock (stock − active reservations), insert checkout_group, orders per supplier (order_number format `ORD-YYYYMMDD-XXXXX`, commission_amount = 4% of subtotal, jsonb address snapshot), snapshot order_items, stock_reservations (expires_at = now+2h), initial order_status_histories, (3) commit, release lock, (4) call payment-service CreateInvoice(amount=grand total, external_id=group_number) → store payments, (5) response invoice_url. Failure at step 4 → mark the group `payment_failed` and release the reservation (compensation), 502 to the user.
- **FR-5.5 Payment-service (separate repo/folder `services/payment`)**: standalone Go gRPC server. RPCs: CreateInvoice, GetInvoice, (CreateDisbursement prepared for phase 6). Integrates the Xendit Invoice API; receives Xendit webhooks on its own HTTP endpoint with callback token verification; forwards events to the monolith via RPC `PaymentEventCallback` (either the monolith exposes a small gRPC server OR payment-service calls the monolith's HTTP webhook `POST /internal/payments/callback` secured with a shared secret — chosen: HTTP callback, simpler; final decision: HTTP callback + `X-Internal-Secret` header). Provide a full mock mode without Xendit for dev.
- **FR-5.6 Paid callback** (idempotent, key: xendit_invoice_id + status): payments.paid → TX: checkout_group paid, all orders paid + history, reservations converted → stock_movements(out) + stock decremented, enqueue email & notification to buyer and each supplier. Duplicate callback doesn't duplicate effects.
- **FR-5.7 order:expire job**: scheduler every minute — pending checkout_group past payments.expires_at → group expired, orders expired + history(actor system), release reservation, notify buyer. Race with payment: transition validated by the state machine (paid can't become expired).
- **FR-5.8 Order state machine** (order usecase, explicit transition table): pending_payment→{paid, expired, cancelled(user)}; paid→{processed(supplier), cancelled(user|supplier → triggers refund via payment-service + restores stock)}; processed→shipped(supplier); shipped→delivered(supplier for own fleet | tracking job for courier — this phase: supplier endpoint marks delivered); delivered→completed(user confirms | autocomplete job after 3 days). Illegal transition → 409 `INVALID_STATUS_TRANSITION`. Every transition writes 1 history row.
- **FR-5.9 Shipment**: `POST /supplier/orders/:id/ship` {method, courier_code?, tracking_number?} — courier requires a tracking number; fleet requires supplier own_fleet_enabled. `GET /user/orders/:id` shows shipment & history timeline.
- **FR-5.10 Order listing**: user `GET /user/orders?status=`, supplier `GET /supplier/orders?status=`, admin `GET /admin/orders` (+filter by supplier/user/date, and `POST /admin/orders/:id/force-status` with audit log for manual intervention).
- **FR-5.11 order:autocomplete job**: delivered > 3 days → completed (actor system). (Supplier balance effect follows in phase 6 — design: an `onOrderCompleted` function called from a single call site.)

### Acceptance Criteria
- AC-5.a E2E (mock RajaOngkir + mock payment): 2 suppliers in 1 cart → preview shows 2 shipping groups → confirm → 2 orders + 1 invoice → paid callback → stock decremented correctly, reservations converted, notifications sent → supplier A ships (courier), supplier B ships (fleet) → delivered → user confirms A completed; B completed via job (tested by backdating delivered_at).
- AC-5.b Two concurrent confirm requests racing for the last unit of stock (parallel test) → exactly one succeeds, one gets 409, no negative stock.
- AC-5.c Unpaid → expire job: reservation released, effective stock restored, order expired. Paid callback sent 2x → single effect.
- AC-5.d Cancel while paid → refund called (mock recorded), stock restored, full history. Illegal transition (pending→shipped) → 409.
- AC-5.e Payment-service runs as a separate process in docker-compose; the monolith still starts (degraded, checkout 502) if the service is down — not a crash.

---

## PHASE 6 — Post-Transaction

**Goal:** trust & money flow: product reviews, full notifications, and auditable ledger-backed supplier payouts.

### Functional Requirements
- **FR-6.1 Settlement on complete**: `onOrderCompleted` → TX: ledger credit_order (subtotal + shipping if own fleet) and debit_commission (4% of subtotal), both with balance_after; update supplier_balances (pending→available per design: simplified — balance goes straight to available on completion). Idempotent per order (unique constraint on ref).
- **FR-6.2 Ledger**: append-only (no update/delete endpoint; DB: revoke UPDATE/DELETE or a blocking trigger). `GET /supplier/ledger?type=&from=&to=` paginated + running balance. Consistency: Σ ledger = balance (proven by test).
- **FR-6.3 Withdraw**: `POST /supplier/withdraws` {amount, bank_account_id} — amount ≥ Rp50,000, ≤ available; TX: hold (available−, pending_withdraw ledger row NOT written yet — hold is enough on balances + request status). `GET /supplier/withdraws`. Cancelable by the supplier only while status is requested.
- **FR-6.4 Approval & disbursement**: `POST /admin/withdraws/:id/approve` → calls payment-service CreateDisbursement → status processing; disbursement-completed callback → status disbursed + ledger debit_withdraw + release hold; failed → restore hold, status failed + notification. `POST /admin/withdraws/:id/reject` {reason} → restores hold. All admin actions → audit log.
- **FR-6.5 Review**: `POST /user/orders/:orderId/items/:itemId/review` {rating 1–5, comment?, image_keys[] ≤3} — only the order owner, status completed, one review per order_item (409). Supplier reply: `POST /supplier/reviews/:id/reply` once. Product rating_avg & rating_count aggregate updated transactionally. Public: `GET /products/:slug/reviews?rating=&cursor=`.
- **FR-6.6 Full notifications**: ensure every event sends in-app + email: order paid/shipped/delivered/completed/cancelled/expired (buyer & supplier per context), new review (supplier), review reply (buyer), withdraw lifecycle (supplier), low stock. Email via the real notification-service (`services/notification`, gRPC, Mailjet templates, mock mode) replacing the phase-2 stub — without changing the usecase.
- **FR-6.7 Lightweight preferences**: `PATCH /user/me/notification-settings` {email_marketing: bool} — transactional emails always sent.

### Acceptance Criteria
- AC-6.a Rp2,000,000 order (courier shipping) completed → ledger: credit 2,000,000, debit 80,000; available = 1,920,000. A second order with own fleet also credits shipping. Running `onOrderCompleted` 2x → single effect.
- AC-6.b Withdraw 1,000,000 → available drops immediately; reject → restored; approve+disbursement succeeds (mock) → ledger debit_withdraw, Σ ledger = balance. Attempting UPDATE on a ledger row via SQL → rejected by the DB.
- AC-6.c Review on a not-yet-completed order → 403; second review on the same item → 409; product rating computed correctly after 3 reviews (avg & count).
- AC-6.d Real notification-service running in compose; stop it → actions still succeed, email goes into Asynq retry (not a failed transaction).

---

## PHASE 7 — Complementary & Hardening

**Goal:** supporting features, reporting, and production readiness.

### Functional Requirements
- **FR-7.1 Wishlist**: `GET/POST/DELETE /user/wishlists` (toggle per product), is_wishlisted flag on product detail when logged in.
- **FR-7.2 Banner**: CRUD `/admin/banners` (media, link, starts/ends schedule, sort) + public `GET /banners` (active & within schedule, cache 5 min).
- **FR-7.3 Supplier report**: `GET /supplier/reports/summary?from=&to=` {gmv, orders_count, aov, top_products[5], sales_per_day[]} and `GET /supplier/reports/export` → enqueue `report:generate` → CSV to storage → notification with download link (presigned, 24h).
- **FR-7.4 Admin report**: `GET /admin/reports/summary` {gmv, commission, orders per status, active suppliers, new users, per-day series} + similar CSV export.
- **FR-7.5 Security hardening**: helmet-equivalent headers, 2MB body limit (except upload), centralized UUID param validation, dependency audit (`govulncheck`), auth-specific rate limit (5/min), lockdown of the internal callback endpoint (secret + optional IP allowlist).
- **FR-7.6 Observability**: `/metrics` Prometheus endpoint (http duration/status, asynq queue depth via exporter), Grafana dashboard JSON in `deploy/`, log sampling for 4xx.
- **FR-7.7 Quality**: usecase coverage ≥70%; phase 5–6 E2E integration test suite run in CI (GitHub Actions: lint, test, build image); full demo seed (`make seed-demo`: 3 approved suppliers, 30 products, 1 order per status) for investor demos.
- **FR-7.8 Documentation**: `docs/openapi.yaml` in sync with all public/user/supplier/admin endpoints; README gets the project running in < 10 minutes from clone; final PROGRESS.md.

### Acceptance Criteria
- AC-7.a CI green end-to-end on the PR. `make seed-demo` + login as 3 roles → the entire demo flow can be walked through without touching the DB manually.
- AC-7.b `/metrics` shows a request histogram; Grafana dashboard shows RPS & p95 during a light load test (k6/hey 100 VU, 1 min) with no 5xx errors and p95 < 300ms for the catalog (external mocked).
- AC-7.c govulncheck clean or every finding documented with a mitigation; internal callback endpoint without secret → 401.
- AC-7.d OpenAPI valid (lint) and covers 100% of registered routes (proven by a route-vs-spec comparison script).

---

## Non-Functional Requirements (apply to all phases)

- **NFR-1** All timestamps `timestamptz` UTC; timezone presentation is the client's concern.
- **NFR-2** Money: `numeric(15,2)` in the DB, integer rupiah in the API (no floats in calculations — use int64 cents OR whole rupiah; decision: whole rupiah int64, no cents).
- **NFR-3** Idempotency required for: payment callback, disbursement callback, settlement, any retryable job.
- **NFR-4** Any list query must be paginated (default 20, max 100) — no unbounded SELECT.
- **NFR-5** N+1 forbidden on list endpoints (prove with query logs during review).
- **NFR-6** Secrets only via env; logs must not contain password/OTP/token/full account number (mask).
- **NFR-7** Every external call (Xendit, RajaOngkir, Mailjet, storage) has an explicit timeout and mapped errors — external failure must not leave internal data inconsistent (use compensation/a simple outbox where needed).

## Definition of Done per Phase

1. All FRs implemented and all ACs pass (automated where possible, manual documented in PROGRESS.md).
2. `make lint && make test` green; migrations up/down clean from an empty database.
3. No TODO/panic placeholder on the production path.
4. PROGRESS.md updated: phase FR & AC checklist checked off + decision notes recorded for any ambiguity.
