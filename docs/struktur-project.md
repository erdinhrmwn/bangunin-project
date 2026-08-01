# Project Structure — Building Materials Marketplace
**Stack:** Golang (Fiber v3) · PostgreSQL · Redis · Asynq · gRPC (Payment/Xendit, Notification/Mailjet) · RajaOngkir

---

## 1. Full Module List

### Core Modules (Monolith)

| # | Module | Description | Primary Role |
|---|--------|-----------|------------|
| 1 | **Auth** | Register, login, refresh token (JWT), email verification (OTP), password reset, logout (token blacklist in Redis) | Everyone |
| 2 | **User** | Profile, change password, address management (multi-address, project/warehouse address), avatar upload | User |
| 3 | **Supplier** | Store registration, verification application (document upload: NIB/KTP), store profile, bank account for payout, operating hours, shipping origin location | Supplier |
| 4 | **Admin** | Supplier approval, category management, user management (ban/suspend), platform commission setting, product & review moderation | Admin |
| 5 | **Category** | Hierarchical categories (parent-child): Cement & Aggregate → Portland Cement, Sand, etc. Slug for SEO | Admin (manage), public (read) |
| 6 | **Product** | Product CRUD, variants (size/unit: sak, batang, m³, dus), multi-image, technical specs (JSON: brand, dimensions, weight), weight & volume per unit (crucial for shipping cost), min. order quantity, draft/active/inactive status | Supplier |
| 7 | **Inventory** | Stock management per variant, stock reservation on checkout (Redis lock + DB), stock movement history, low-stock notifications | Supplier |
| 8 | **Search & Catalog** | Full-text search (PostgreSQL `tsvector` + GIN index), filters (category, price, supplier location, rating, brand), sorting, cursor-based pagination | Public |
| 9 | **Cart** | Per-user cart (persisted in DB, cached in Redis), item grouping per supplier, stock & min. order validation on load | User |
| 10 | **Checkout** | Order summary, automatic split order per supplier, shipping cost calculation (RajaOngkir: weight + volumetric), courier selection, price & stock re-validation, order creation + stock reservation (transactional) | User |
| 11 | **Order** | Order status state machine (pending_payment → paid → processed → shipped → delivered → completed / cancelled / refunded), status history, cancellation, receipt confirmation, auto-complete after N days | User, Supplier, Admin |
| 12 | **Shipping** | RajaOngkir integration (cost & waybill tracking), tracking number entry by supplier, **supplier's own fleet delivery** option (flat rate/per km — common for heavy materials like sand & steel), supplier shipping coverage zone | Supplier |
| 13 | **Review & Rating** | Per-product review (only buyers whose order is completed), aggregate rating on product & store, supplier reply, review photos | User |
| 14 | **Wishlist** | Save favorite products | User |
| 15 | **Payout / Settlement** | Escrow flow: funds held until order completed → supplier balance → withdraw request → admin approve → disbursement (via payment service). Supplier transaction ledger, platform commission deduction | Supplier, Admin |
| 16 | **Notification (in-app)** | In-app notifications (order update, approval, payout) + email trigger via gRPC to notification service. Read/unread status | Everyone |
| 17 | **Report & Analytics** | Supplier dashboard (sales, best-selling products), admin dashboard (GMV, commission, active suppliers), CSV export | Supplier, Admin |
| 18 | **Media / File Upload** | Upload product images & verification documents to object storage (S3/MinIO), presigned URL, type & size validation, resize/compress via worker | Everyone |
| 19 | **Audit Log** | Logging of sensitive actions (approval, price changes, user ban, payout) for admin audit purposes | Admin |
| 20 | **Banner / Promo Display** | Homepage banner & featured products managed by admin (not a voucher system — vouchers remain deferred per earlier decision) | Admin |

### External Services (gRPC — already in the design)

| Service | Responsibility |
|---------|----------------|
| **Payment Service (Xendit)** | Create invoice/VA/e-wallet, receive Xendit webhook, disbursement for supplier payout, refund |
| **Notification Service (Mailjet)** | Send transactional email (verification, order, payout) based on templates |

### Background Jobs (Asynq Worker)

- `email:send` — delegate email sending via notification service
- `order:expire` — cancel orders unpaid within X hours + release stock reservation
- `order:autocomplete` — auto-complete order N days after delivered
- `payment:sync` — reconcile payment status with payment service
- `stock:release` — release expired stock reservations
- `media:process` — resize/compress images after upload
- `report:generate` — generate large reports/CSV exports
- `notification:lowstock` — check & send low-stock notifications (scheduled/cron via Asynq Scheduler)

---

## 2. Concrete Directory Structure

```
marketplace-bahan-bangunan/
├── cmd/
│   ├── api/main.go                  # HTTP server (Fiber v3)
│   ├── worker/main.go               # Asynq worker + scheduler
│   └── migrate/main.go              # migration runner + seeder
│
├── config/
│   ├── config.go                    # config struct + Viper loader
│   └── config.yaml                  # default config (overridden by .env)
│
├── internal/
│   ├── domain/                      # LAYER 1: entities + contracts (pure Go, no external dependency)
│   │   ├── entity/
│   │   │   ├── user.go
│   │   │   ├── supplier.go
│   │   │   ├── category.go
│   │   │   ├── product.go           # + ProductVariant, ProductImage
│   │   │   ├── inventory.go         # StockMovement, StockReservation
│   │   │   ├── cart.go
│   │   │   ├── order.go             # Order, OrderItem, OrderStatusHistory
│   │   │   ├── shipment.go
│   │   │   ├── review.go
│   │   │   ├── payout.go            # SupplierBalance, LedgerEntry, WithdrawRequest
│   │   │   ├── notification.go
│   │   │   └── audit_log.go
│   │   ├── repository/              # repository interface per domain
│   │   │   ├── user_repository.go
│   │   │   ├── product_repository.go
│   │   │   ├── order_repository.go
│   │   │   └── ... (per domain)
│   │   ├── service/                 # external service interface
│   │   │   ├── payment_service.go   # contract for payment gRPC
│   │   │   ├── notification_service.go
│   │   │   ├── shipping_gateway.go  # RajaOngkir contract
│   │   │   └── storage_service.go   # object storage contract
│   │   └── errs/
│   │       └── errors.go            # domain error types (ErrNotFound, ErrInsufficientStock, ...)
│   │
│   ├── usecase/                     # LAYER 2: business logic
│   │   ├── auth/
│   │   ├── user/
│   │   ├── supplier/
│   │   ├── admin/
│   │   ├── category/
│   │   ├── product/
│   │   ├── inventory/
│   │   ├── catalog/                 # search, filter, public product listing
│   │   ├── cart/
│   │   ├── checkout/                # orchestration: validate → shipping cost → split order → payment
│   │   ├── order/
│   │   ├── shipping/
│   │   ├── review/
│   │   ├── wishlist/
│   │   ├── payout/
│   │   ├── notification/
│   │   ├── report/
│   │   └── media/
│   │
│   ├── delivery/                    # LAYER 3: transport
│   │   └── http/
│   │       ├── handler/             # 1 file per module, mapped to usecase
│   │       ├── middleware/
│   │       │   ├── auth.go          # JWT parse + inject claims into context
│   │       │   ├── rbac.go          # RequireRole("admin"|"supplier"|"user")
│   │       │   ├── ratelimit.go     # sliding window via Redis
│   │       │   ├── recover.go
│   │       │   ├── requestid.go
│   │       │   └── logger.go        # structured request log
│   │       ├── dto/                 # request/response structs + validation tags per module
│   │       └── route/
│   │           ├── route.go         # main registration
│   │           ├── public.go        # /api/v1/catalog, /auth
│   │           ├── user.go          # /api/v1/user/* (JWT + user role)
│   │           ├── supplier.go      # /api/v1/supplier/*
│   │           └── admin.go         # /api/v1/admin/*
│   │
│   ├── repository/                  # LAYER 4: persistence implementation
│   │   ├── postgres/                # pgx + sqlc (or GORM per team preference)
│   │   │   └── ... (per domain)
│   │   └── redis/
│   │       ├── cache.go             # product/category cache
│   │       ├── session.go
│   │       └── stock_lock.go        # distributed lock for stock reservation
│   │
│   ├── infra/                       # adapters to the outside world
│   │   ├── database/postgres.go     # pool + health
│   │   ├── redis/client.go
│   │   ├── grpcclient/
│   │   │   ├── payment.go           # implementation of domain/service/payment_service.go
│   │   │   └── notification.go
│   │   ├── rajaongkir/client.go
│   │   ├── storage/s3.go            # S3/MinIO
│   │   └── queue/
│   │       ├── client.go            # Asynq client (enqueue)
│   │       ├── tasks.go             # task type + payload definitions
│   │       └── handlers/            # per-task handler (used by cmd/worker)
│   │
│   └── app/
│       ├── server.go                # Fiber wiring: global middleware, routes, graceful shutdown
│       └── container.go             # manual dependency injection (or wire/fx)
│
├── pkg/                             # reusable utilities, domain-agnostic
│   ├── jwt/
│   ├── response/                    # standard envelope {success, message, data, meta, errors}
│   ├── pagination/                  # cursor & offset helper
│   ├── validator/                   # go-playground/validator + custom rules (ID phone, etc.)
│   ├── hash/                        # bcrypt/argon2
│   ├── logger/                      # zerolog/zap setup
│   └── apperr/                      # domain error → HTTP status mapping
│
├── proto/                           # payment & notification .proto + generated code
│   ├── payment/v1/payment.proto
│   └── notification/v1/notification.proto
│
├── migrations/                      # golang-migrate: 000001_create_users.up.sql / .down.sql
├── seeds/                           # initial data: roles, building-materials categories, default admin
├── docs/
│   ├── openapi.yaml                 # Swagger/OpenAPI spec
│   └── erd.md                       # ERD + Mermaid diagrams from earlier planning
├── deploy/
│   ├── docker-compose.yml           # postgres, redis, minio, api, worker
│   ├── Dockerfile.api
│   └── Dockerfile.worker
├── scripts/                         # helpers: gen-proto.sh, lint.sh
├── test/
│   ├── integration/                 # tests with testcontainers
│   └── mocks/                       # generated mocks (mockery)
├── .env.example
├── .golangci.yml
├── Makefile                         # run, migrate, seed, gen-proto, test, lint
└── go.mod
```

---

## 3. Key Design Decisions

**Split order per supplier.** A single checkout can contain products from multiple suppliers → split into multiple `orders` (one per supplier) under a single `checkout_group_id`, but **a single payment invoice** to Xendit. Shipping cost is calculated per order (each supplier has a different origin).

**Weight & volumetric are mandatory on products.** Building materials are heavy (40kg cement sacks, long steel bars). Shipping cost = max(actual weight, volumetric weight). For items impractical to ship via courier (sand per m³), enable the *supplier's own fleet delivery* option with a coverage zone.

**Stock reservation, not immediate deduction.** When an order is created: stock is reserved (`stock_reservations` row + Redis lock for race conditions). Paid → reservation converts into a stock deduction. Expired → the `order:expire` job releases the reservation.

**Escrow & ledger for payout.** Funds go to the platform first. Order completed → credited to supplier balance (recorded in `ledger_entries`, platform commission debited). Withdraw → admin approval → disbursement via payment service. Ledger is append-only so balance can always be audited.

**Order status as a state machine.** Valid transitions are explicitly defined in the usecase (e.g., `paid → processed` only by supplier, `cancelled` only from `pending_payment`/`paid` with refund). Every transition is recorded in `order_status_histories`.

**RBAC via route group + middleware.** Since `role_id` is already a direct FK on `users`, this suffices: `admin := v1.Group("/admin", mw.Auth(), mw.RequireRole("admin"))`. No permission table needed yet — can evolve later if granular permissions are required.

---

## 4. Recommended Implementation Phase Order

1. **Foundation** — project skeleton, config, database + initial migrations, Redis, logger, response envelope, error handling, basic middleware, Makefile, docker-compose.
2. **Auth + RBAC** — register/login/refresh, email verification (via worker + notification service stub), role middleware.
3. **Supplier onboarding** — store registration, document upload (media module), admin approval.
4. **Catalog** — category, product + variant + image, inventory, public search & filter.
5. **Core transactions** — cart → checkout (RajaOngkir shipping cost, split order) → payment service integration → order state machine → shipping/tracking number.
6. **Post-transaction** — review, in-app notification, payout/ledger + withdraw.
7. **Complementary** — wishlist, report/analytics, audit log, banner, hardening (rate limit, integration tests, light load test).

Each phase produces something demoable, and dependencies between phases flow in one direction.

---

## 5. Fiber v3 Notes

- Fiber v3 is still a release candidate — the API may change minor versions until stable. Pin the version in `go.mod` and read the changelog on every upgrade.
- Key changes from v2: `fiber.Ctx` becomes an interface, binding via `c.Bind().Body(&dto)`, built-in middleware restructured.
- Since all Fiber code is locked into `internal/delivery/http`, the risk of breaking changes is isolated to one layer — usecase and repository are untouched.
