# Requirements per Phase — Building Materials Marketplace

Implementation reference document. Each phase contains: goal, scope, functional requirements (FR), endpoints, migrations, background jobs, and acceptance criteria (AC) that must pass before the phase is considered done. Saved as `docs/requirements.md` in the repo — CLAUDE.md refers to this document.

Convention: `FR-x.y` = functional requirement, phase x, number y. All endpoints are under prefix `/api/v1`. All responses use the standard envelope `{success, message, data, meta, errors}`.

---

## PHASE 1 — Foundation

**Goal:** repo can be cloned, `docker compose up`, migrate, seed, and serve a health endpoint with the whole backbone (config, DB, Redis, logger, middleware, error handling) ready for the next phase.

**In scope:** skeleton, tooling, local infrastructure, initial migrations, basic middleware.
**Out of scope:** all business features, gRPC client, RajaOngkir, storage.

### Functional Requirements
- **FR-1.1 Project structure** per CLAUDE.md §3. Every folder contains at least a `doc.go` or a real file — no unused empty folders.
- **FR-1.2 Config**: Viper reads `config.yaml` + env override (prefix `APP_`). Typed config struct: App{Name, Env, Port}, DB{DSN, MaxConns...}, Redis, JWT{Secret, AccessTTL, RefreshTTL}, and empty sections for Xendit/RajaOngkir/R2 (filled in in their respective phases). App fails fast with a clear message if required config is empty.
- **FR-1.3 Database**: pgxpool connection with ping on startup, graceful close on shutdown. Migrations via golang-migrate run through `cmd/migrate` and the Makefile target.
- **FR-1.4 Initial migrations**: `roles` (seed: admin, supplier, user) and `users` per the ERD (uuid v7 default, role_id FK, unique email, index on email & status). Both up and down work.
- **FR-1.5 Seeder**: idempotent (safe to run repeatedly). Seeds roles + 1 default admin account from env (`SEED_ADMIN_EMAIL`, `SEED_ADMIN_PASSWORD`).
- **FR-1.6 Redis**: client with ping on startup; get/set/del helper with TTL in `internal/repository/redis`.
- **FR-1.7 Logger**: zerolog JSON to stdout; level from config; standard fields (service, env). Request log middleware records method, path, status, latency, request_id.
- **FR-1.8 Basic middleware**: recover (panic → 500 envelope + stack log), request ID (header `X-Request-ID`, generated if empty), configurable CORS, Redis sliding window rate limit (default 60 req/minute/IP, path-exempt: /health).
- **FR-1.9 Response & error**: `pkg/response` (Success, Error, paginated meta) and `pkg/apperr` — AppError{Code, Message, HTTPStatus} type + mapper from domain errs (`ErrNotFound`→404, `ErrValidation`→422, `ErrUnauthorized`→401, `ErrForbidden`→403, `ErrConflict`→409, default 500 without leaking internal details).
- **FR-1.10 Health**: `GET /health` → app status, db (ping), redis (ping), version, uptime. 503 if a dependency is down.
- **FR-1.11 Worker skeleton**: `cmd/worker` runs an Asynq server connected to Redis with 1 dummy task `system:heartbeat` registered; scheduler ready but empty.
- **FR-1.12 Tooling**: Makefile (run, run-worker, build, migrate-up/down/create, seed, test, lint, gen-proto placeholder), `.golangci.yml`, complete `.env.example`, docker-compose (postgres 16, redis 7, minio, api, worker) with healthcheck and persistent volumes.
- **FR-1.13 Graceful shutdown**: SIGINT/SIGTERM → stop accepting requests, wait for in-flight requests max 10 sec, close DB/Redis.

### Acceptance Criteria
- AC-1.a `docker compose up -d` → all containers healthy; `make migrate-up && make seed` succeeds; `curl /health` = 200 with db & redis status "ok".
- AC-1.b Kill Redis → /health = 503; forced panic in a test handler → clean 500 envelope + stack in log, process doesn't die.
- AC-1.c `make lint` and `make test` pass; migrate down then up again succeeds; seed run twice without error/duplicates.
- AC-1.d Send 70 requests/minute to a test endpoint → some rejected with 429 in the standard envelope.

---

## PHASE 2 — Auth + RBAC

**Goal:** full account lifecycle: register, verify email, login/refresh/logout, reset password, and per-role route protection.

**In scope:** user & supplier-account auth, JWT, OTP email (via stub notification), RBAC middleware.
**Out of scope:** supplier/store profile (phase 3), OAuth/social login.

### Functional Requirements
- **FR-2.1 Register** `POST /auth/register` {name, email, password, phone, role: user|supplier}. Password min 8 chars with a digit; email unique (409 if already exists); admin role CANNOT be registered via this endpoint. Password hashed with bcrypt cost 12. Account status `active`, `email_verified_at` null.
- **FR-2.2 Email verification**: after register, generate a 6-digit OTP, store in Redis `otp:verify:{userID}` TTL 10 minutes, enqueue `email:send`. `POST /auth/verify-email` {email, otp} → set email_verified_at. `POST /auth/resend-otp` rate-limited 1x/minute per email. 5 wrong OTP attempts → OTP invalidated.
- **FR-2.3 Login** `POST /auth/login` {email, password} → access token (JWT, TTL 15 min, claims: sub, role, jti) + refresh token (random opaque, TTL 7 days, stored in Redis `refresh:{token}`→userID). Email not verified → 403 with special code `EMAIL_NOT_VERIFIED`. Suspended/banned account → 403. Brute force guard: 10 failures/15 minutes per email+IP → temporary block.
- **FR-2.4 Refresh** `POST /auth/refresh` {refresh_token} → rotation: old token deleted, new pair issued. Unknown token → 401.
- **FR-2.5 Logout** `POST /auth/logout` (auth) → delete refresh token + blacklist access token jti in Redis until expiry.
- **FR-2.6 Reset password**: `POST /auth/forgot-password` (always returns 200 to avoid leaking whether an email exists; sends OTP if it does) → `POST /auth/reset-password` {email, otp, new_password} → invalidates all of the user's refresh tokens.
- **FR-2.7 Auth Middleware**: parse Bearer JWT, reject expired/blacklisted/invalid (401), inject claims into context (`ctxutil.UserID(c)`, `ctxutil.Role(c)`).
- **FR-2.8 RequireRole(roles...) Middleware**: 403 if role doesn't match. Route groups `/user`, `/supplier`, `/admin` set up and proven with a `GET {group}/me` endpoint.
- **FR-2.9 Basic profile**: `GET /user/me`, `PATCH /user/me` (name, phone), `PATCH /user/me/password` (verifies old password).
- **FR-2.10 Notification stub**: log-only implementation of `domain/service.NotificationService` (prints email content to log) in `infra/grpcclient` — used until the real service is available; swap the implementation without changing the usecase.

### Acceptance Criteria
- AC-2.a Happy path flow: register→OTP (read from stub log)→verify→login→access `/user/me`→refresh→logout→old access token rejected with 401.
- AC-2.b User with `user` role accessing `/supplier/me` → 403; no token → 401; expired token → 401.
- AC-2.c A rotated refresh token reused → 401. Password reset invalidates all old sessions.
- AC-2.d Auth usecase unit tests cover: duplicate email, wrong/expired/5x-failed OTP, brute force lock, admin role rejected at registration.

---

## PHASE 3 — Supplier Onboarding

**Goal:** supplier accounts complete their store profile, upload documents, and get approved/rejected by admin with an audit trail.

**In scope:** store profile, KYC documents, bank account, media upload (MinIO/R2), admin approval, audit log, status notifications.
**Out of scope:** products (phase 4).

### Functional Requirements
- **FR-3.1 Media module**: `POST /media/upload` (auth; multipart) → validate type (jpg/png/webp/pdf) & size (images ≤5MB, pdf ≤10MB) → store in bucket with key `{scope}/{uuid}.{ext}` → return url + key. Enqueue `media:process` to resize images (max 1600px) — worker replaces the original file. `domain/service.StorageService` implementation based on the S3 API (local MinIO, production R2).
- **FR-3.2 Store profile**: `POST /supplier/profile` & `PUT /supplier/profile` {store_name, description, origin_city_id, pickup_address, own_fleet_enabled, fleet_coverage_km?, fleet_flat_rate?}. Slug uniquely generated from store_name. origin_city_id must be valid (for this phase an integer > 0 is enough; RajaOngkir validation follows in phase 5). One user maximum one supplier (409).
- **FR-3.3 Documents**: `POST /supplier/documents` {doc_type: nib|ktp|npwp, file_key} — max 1 active document per type (re-uploading replaces it, status returns to pending). `GET /supplier/documents`.
- **FR-3.4 Bank account**: CRUD `/supplier/bank-accounts`; exactly one is_default; account number 6–20 digits.
- **FR-3.5 Submit application**: `POST /supplier/submit` — only allowed if profile is complete + at least KTP & NIB documents exist. Supplier status: `draft → pending`. Cannot submit while already pending/approved.
- **FR-3.6 Admin review**: `GET /admin/suppliers?status=&q=&page=` (list + filter), `GET /admin/suppliers/:id` (detail + documents + presigned url), `POST /admin/suppliers/:id/approve` (sets approved + verified_at), `POST /admin/suppliers/:id/reject` {reason} (reason required; supplier can fix and resubmit), `POST /admin/suppliers/:id/suspend` {reason}.
- **FR-3.7 Audit log**: every approve/reject/suspend recorded in audit_logs {actor_id, action, entity, metadata: {reason, before, after}, ip}. `GET /admin/audit-logs` with action & date range filters.
- **FR-3.8 Notifications**: supplier status change → in-app notification (notifications table) + enqueue email. `GET /user/notifications` + `POST /user/notifications/:id/read` + unread count.
- **FR-3.9 Guard**: every selling-related `/supplier/*` endpoint in later phases must pass through the `RequireApprovedSupplier` middleware (403 `SUPPLIER_NOT_APPROVED` if not yet approved) — middleware built in this phase.

### Acceptance Criteria
- AC-3.a Full flow: supplier completes profile → uploads 2 documents → submits → admin rejects with a reason → supplier receives notification + reason → fixes → resubmits → approve → notification received, verified_at set, audit_logs contains 2 entries.
- AC-3.b Uploading a .exe file or 20MB → 422. Large images get resized by the worker (verify resulting dimensions).
- AC-3.c Submit without NIB → 422 with a list of missing items. Accessing a (dummy) selling endpoint before being approved → 403.
- AC-3.d New migrations: suppliers, supplier_documents, supplier_bank_accounts, notifications, audit_logs — clean up/down.

---

## PHASE 4 — Catalog

**Goal:** approved suppliers manage a full product catalog; the public can quickly search and browse the catalog.

**In scope:** categories, products+variants+images, stock & movements, public listing + full-text search, cache.
**Out of scope:** cart/checkout (phase 5), reviews (phase 6).

### Functional Requirements
- **FR-4.1 Categories (admin)**: hierarchical CRUD `/admin/categories` (parent_id, max 3 levels), unique slug, is_active, sort_order. Delete only allowed if no children & no products (409). Public: `GET /categories` (tree, Redis cache 10 minutes, invalidated on mutation). Seed ±20 realistic building-material categories (Cement & Aggregate, Iron & Steel, Wood, Paint, Ceramic & Granite, Sanitary, Roofing, Electrical, Piping, etc.).
- **FR-4.2 Products (supplier, approved-only)**: CRUD `/supplier/products`. Fields: category_id, name, description, specs (free-form jsonb: brand, material, dimensions), status draft|active|inactive. Publish (draft→active) only if ≥1 active variant and ≥1 image. Globally unique slug.
- **FR-4.3 Variants**: nested CRUD `/supplier/products/:id/variants`. Required fields: name, sku (unique per supplier), unit (enum: sak|batang|m3|dus|pcs|lembar|kg|roll|set), price>0, **weight_gram>0**, dimensions in cm (may be 0 for non-volumetric items), min_order_qty≥1, stock≥0. Updating price doesn't change existing orders (snapshot — enforced in phase 5).
- **FR-4.4 Images**: attached from media upload, max 8 per product, exactly one is_primary, sortable.
- **FR-4.5 Stock**: `POST /supplier/variants/:id/stock-adjustment` {qty(+/-), note} → writes stock_movements (type in/out/adjustment, stock_after) in a transaction together with the stock update; stock cannot go negative (409). `GET /supplier/variants/:id/movements` (history, paginated).
- **FR-4.6 Public catalog**: `GET /products?q=&category=&min_price=&max_price=&city=&sort=(newest|price_asc|price_desc|rating)&cursor=` — only active products from approved suppliers. q uses tsvector (name+description+specs brand) websearch_to_tsquery, ranked with ts_rank. Concise response: id, slug, name, primary_image, price_min–max, unit, supplier{name, city}, rating. Cursor-based pagination.
- **FR-4.7 Public detail**: `GET /products/:slug` → full active variants + available stock (stock − active reservations; before phase 5, reservations=0), images, specs, brief supplier profile. Redis cache 60 sec, invalidated when supplier updates the product.
- **FR-4.8 search_vector** maintained by a PostgreSQL trigger (INSERT/UPDATE), GIN index. EXPLAIN uses the index.
- **FR-4.9 Public store**: `GET /suppliers/:slug` → profile + its products (paginated).
- **FR-4.10 Low-stock job**: daily scheduler `notification:lowstock` — variant stock < threshold (default 5, configurable per supplier later; hardcoded default for now) → notify supplier, max 1x/day/variant.

### Acceptance Criteria
- AC-4.a Supplier creates product "Semen Portland 40kg" with 2 variants + 3 images → publish → appears in `GET /products?q=semen`, and searching "portland 40" also finds it.
- AC-4.b Publishing without images → 422. Variant without weight_gram → 422. Duplicate SKU within one supplier → 409.
- AC-4.c Adjustment of −10 on stock of 5 → 409; +10 then −3 → 2 movement rows with correct stock_after.
- AC-4.d Suspended supplier's products don't appear in public catalog. Cache: changing product name → public detail updates ≤ invalidation time (not waiting for the full 60 sec).
- AC-4.e Repository search integration test with testcontainers: seeding 50 products, query q+filter+sort returns correct results and ordering.

---

## PHASE 5 — Core Transactions

**Goal:** end-to-end money and goods flow: cart → checkout (shipping) → pay via payment-service (Xendit) → order runs through a state machine → shipped → received.

**In scope:** cart, checkout preview/confirm, RajaOngkir client, stock reservation + Redis lock, payment-service gRPC (actually built), webhook, order lifecycle, shipment, expire job.
**Out of scope:** reviews, payout (phase 6).

### Functional Requirements
- **FR-5.1 Cart**: `GET /user/cart`, `POST /user/cart/items` {variant_id, qty}, `PATCH .../items/:id`, `DELETE .../items/:id`. Merges qty for the same variant. Response grouped per supplier, each item carries an issue flag: out_of_stock, below_min_order, price_changed(since added), product_inactive. Problematic items don't block viewing the cart but do block checkout.
- **FR-5.2 RajaOngkir client** (`infra/rajaongkir`): implements `domain/service.ShippingGateway` — GetCost(originCityID, destCityID, weightGram, couriers[]) and (stub for this phase) TrackWaybill. 5 sec timeout, 1 retry, error → `ErrShippingUnavailable`. API key configured via env; provide a mock mode (`RAJAONGKIR_MOCK=true`) returning deterministic rates for dev/test.
- **FR-5.3 Checkout preview** `POST /user/checkout/preview` {address_id, items[]}: validate all items (stock, min order, active, supplier approved) → group per supplier → per group compute chargeable weight = Σ max(weight, volumetric l·w·h/6000) → call shipping cost in parallel per group → return: groups[{supplier, items, subtotal, shipping_options[courier/service/etd/cost + supplier fleet option if enabled & distance within coverage]}], grand_total_estimate. Address must belong to the user.
- **FR-5.4 Checkout confirm** `POST /user/checkout/confirm` {address_id, items[], shipping_choice per group}: atomic sequence — (1) Redis lock `stock:{variantID}` (SETNX+TTL 10 sec, sorted by ID to avoid deadlock), (2) DB TX: re-validate price & effective stock (stock − active reservations), insert checkout_group, orders per supplier (order_number format `ORD-YYYYMMDD-XXXXX`, commission_amount = 4% of subtotal, jsonb address snapshot), order_items snapshot, stock_reservations (expires_at = now+2 hours), initial order_status_histories, (3) commit, release lock, (4) call payment-service CreateInvoice(amount=grand total, external_id=group_number) → store payments, (5) response invoice_url. Failure in step 4 → mark group `payment_failed` and release reservation (compensation), 502 to user.
- **FR-5.5 Payment-service (separate repo/folder `services/payment`)**: standalone Go gRPC server. RPCs: CreateInvoice, GetInvoice, (CreateDisbursement prepared for phase 6). Xendit Invoice API integration; receives Xendit webhook on its own HTTP endpoint with callback token verification; forwards the event to the monolith via RPC `PaymentEventCallback` (either the monolith exposes a small gRPC server OR payment-service calls the monolith's HTTP webhook `POST /internal/payments/callback` secured with a shared secret — chosen: HTTP callback, simpler; final decision: HTTP callback + `X-Internal-Secret` header). Provide a full mock mode without Xendit for dev.
- **FR-5.6 Paid callback** (idempotent, key: xendit_invoice_id + status): payments.paid → TX: checkout_group paid, all orders paid + history, convert reservation → stock_movements(out) + reduce stock, enqueue email & notification to buyer and each supplier. Duplicate callback doesn't duplicate the effect.
- **FR-5.7 order:expire Job**: scheduler every minute — pending checkout_group past payments.expires_at → group expired, orders expired + history(actor system), release reservation, notify buyer. Race with payment: transitions validated by the state machine (paid cannot become expired).
- **FR-5.8 Order state machine** (order usecase, explicit transition table): pending_payment→{paid, expired, cancelled(user)}; paid→{processed(supplier), cancelled(user|supplier → triggers refund via payment-service + returns stock)}; processed→shipped(supplier); shipped→delivered(supplier for own fleet | tracking job for courier — this phase: supplier endpoint marks delivered); delivered→completed(user confirm | autocomplete job after 3 days). Illegal transition → 409 `INVALID_STATUS_TRANSITION`. Every transition writes 1 history row.
- **FR-5.9 Shipment**: `POST /supplier/orders/:id/ship` {method, courier_code?, tracking_number?} — tracking number required for courier; own fleet requires supplier own_fleet_enabled. `GET /user/orders/:id` shows shipment & history timeline.
- **FR-5.10 Order listing**: user `GET /user/orders?status=`, supplier `GET /supplier/orders?status=`, admin `GET /admin/orders` (+filter by supplier/user/date, and `POST /admin/orders/:id/force-status` with audit log for intervention).
- **FR-5.11 order:autocomplete Job**: delivered > 3 days → completed (actor system). (Supplier balance effect follows in phase 6 — event design: a single `onOrderCompleted` function called from one place.)

### Acceptance Criteria
- AC-5.a E2E (mock RajaOngkir + mock payment): 2 suppliers in 1 cart → preview shows 2 shipping groups → confirm → 2 orders + 1 invoice → callback paid → stock correctly reduced, reservation converted, notifications sent → supplier A ships (courier), supplier B ships (own fleet) → delivered → user confirms completed for A; B completed via job (test by backdating delivered_at).
- AC-5.b Two simultaneous confirm requests competing for the last stock (parallel test) → exactly one succeeds, one gets 409, no negative stock.
- AC-5.c Unpaid → expire job: reservation released, effective stock restored, order expired. Callback paid sent 2x → single effect.
- AC-5.d Cancel while paid → refund called (mock recorded), stock returned, full history. Illegal transition (pending→shipped) → 409.
- AC-5.e Payment-service runs as a separate process in docker-compose; monolith still starts (degraded, checkout returns 502) if the service is down — no crash.

---

## PHASE 6 — Post-transaction

**Goal:** trust & money flow: product reviews, complete notifications, and auditable ledger-based supplier payouts.

### Functional Requirements
- **FR-6.1 Settlement on complete**: `onOrderCompleted` → TX: ledger credit_order (subtotal + shipping if own fleet) and debit_commission (4% of subtotal), both with balance_after; update supplier_balances (pending→available per design: simplified — incoming balance goes straight to available upon completion). Idempotent per order (unique constraint ref).
- **FR-6.2 Ledger**: append-only (no update/delete endpoint; DB: revoke UPDATE/DELETE or block via trigger). `GET /supplier/ledger?type=&from=&to=` paginated + running balance. Consistency: Σ ledger = balance (proven by test).
- **FR-6.3 Withdraw**: `POST /supplier/withdraws` {amount, bank_account_id} — amount ≥ Rp50.000, ≤ available; TX: hold (available−, pending_withdraw row NOT yet written to ledger — hold is enough in balances + request status). `GET /supplier/withdraws`. Cancellable by supplier only while status is requested.
- **FR-6.4 Approval & disbursement**: `POST /admin/withdraws/:id/approve` → calls payment-service CreateDisbursement → status processing; disbursement completed callback → status disbursed + ledger debit_withdraw + release hold; failed → return hold, status failed + notification. `POST /admin/withdraws/:id/reject` {reason} → return hold. All admin actions → audit log.
- **FR-6.5 Review**: `POST /user/orders/:orderId/items/:itemId/review` {rating 1–5, comment?, image_keys[] ≤3} — only the order owner with status completed, one review per order_item (409). Supplier reply: `POST /supplier/reviews/:id/reply` once. Product rating_avg & rating_count aggregate updated transactionally. Public: `GET /products/:slug/reviews?rating=&cursor=`.
- **FR-6.6 Complete notifications**: ensure every event sends in-app + email: order paid/shipped/delivered/completed/cancelled/expired (buyer & supplier as relevant), new review (supplier), review reply (buyer), withdraw lifecycle (supplier), low stock. Email via the real notification-service (`services/notification`, gRPC, Mailjet templates, mock mode) replacing the phase 2 stub — without changing the usecase.
- **FR-6.7 Light preferences**: `PATCH /user/me/notification-settings` {email_marketing: bool} — transactional emails always sent.

### Acceptance Criteria
- AC-6.a Order of Rp2.000.000 (courier shipping) completed → ledger: credit 2.000.000, debit 80.000; available = 1.920.000. Second order with own fleet → shipping also credited. Running `onOrderCompleted` 2x → single effect.
- AC-6.b Withdraw 1.000.000 → available drops immediately; reject → returned; approve+disbursement success (mock) → ledger debit_withdraw, Σ ledger = balance. Attempting to UPDATE a ledger row via SQL → rejected by DB.
- AC-6.c Review on a not-yet-completed order → 403; second review on the same item → 409; product rating correctly computed after 3 reviews (avg & count).
- AC-6.d Real notification-service running in compose; turn it off → actions still succeed, email goes into Asynq retry (not a transaction failure).

---

## PHASE 7 — Extras & Hardening

**Goal:** supporting features, reports, and production readiness.

### Functional Requirements
- **FR-7.1 Wishlist**: `GET/POST/DELETE /user/wishlists` (toggle per product), is_wishlisted flag in product detail when logged in.
- **FR-7.2 Banner**: CRUD `/admin/banners` (media, link, starts/ends schedule, sort) + public `GET /banners` (active & within schedule, cache 5 minutes).
- **FR-7.3 Supplier report**: `GET /supplier/reports/summary?from=&to=` {gmv, orders_count, aov, top_products[5], sales_per_day[]} and `GET /supplier/reports/export` → enqueue `report:generate` → CSV to storage → download link notification (presigned, 24 hours).
- **FR-7.4 Admin report**: `GET /admin/reports/summary` {gmv, commission, orders per status, active suppliers, new users, per-day series} + similar CSV export.
- **FR-7.5 Security hardening**: helmet-equivalent headers, body limit 2MB (except uploads), centralized UUID param validation, dependency audit (`govulncheck`), auth-specific rate limit (5/minute), internal callback endpoint lockdown (secret + optional IP allowlist).
- **FR-7.6 Observability**: `/metrics` Prometheus endpoint (http duration/status, asynq queue depth via exporter), Grafana dashboard JSON in `deploy/`, log sampling for 4xx.
- **FR-7.7 Quality**: usecase coverage ≥70%; phase 5–6 E2E integration test suite run in CI (GitHub Actions: lint, test, build image); full demo seed (`make seed-demo`: 3 approved suppliers, 30 products, 1 order per status) for investor demo.
- **FR-7.8 Documentation**: `docs/openapi.yaml` synced with all public/user/supplier/admin endpoints; README gets the project running < 10 minutes from clone; final PROGRESS.md.

### Acceptance Criteria
- AC-7.a CI green end-to-end on PR. `make seed-demo` + login with 3 roles → the entire demo flow can be shown without touching the DB manually.
- AC-7.b `/metrics` shows a request histogram; Grafana dashboard shows RPS & p95 during a light load test (k6/hey 100 VU, 1 minute) with no 5xx errors and p95 < 300ms for the catalog (external mocked).
- AC-7.c govulncheck clean or all findings recorded with mitigation; internal callback endpoint without secret → 401.
- AC-7.d OpenAPI valid (lint) and covers 100% of registered routes (proven by a script comparing routes vs spec).

---

## Non-Functional Requirements (apply to all phases)

- **NFR-1** All timestamps `timestamptz` UTC; timezone presentation is the client's concern.
- **NFR-2** Money: `numeric(15,2)` in DB, integer rupiah in the API (no floats in calculations — use int64 cents OR whole rupiah; decision: whole rupiah int64, no cents).
- **NFR-3** Idempotency required for: payment callback, disbursement callback, settlement, any retryable job.
- **NFR-4** Any list query must be paginated (default 20, max 100) — no unbounded SELECT.
- **NFR-5** N+1 queries are forbidden on list endpoints (prove with query logs during review).
- **NFR-6** Secrets only via env; logs must not contain password/OTP/token/full account number (mask).
- **NFR-7** All external calls (Xendit, RajaOngkir, Mailjet, storage) have an explicit timeout and mapped errors — external failures must not leave internal data inconsistent (use compensation/a simple outbox if needed).

## Definition of Done per Phase

1. All FRs implemented and all ACs pass (automated where possible, manual documented in PROGRESS.md).
2. `make lint && make test` pass; migrate up/down clean from an empty database.
3. No TODO/panic placeholders on the production path.
4. PROGRESS.md updated: phase FR & AC checklist checked off + notes on decisions made for any ambiguity.
