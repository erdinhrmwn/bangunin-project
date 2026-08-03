# Struktur Project — Marketplace Bahan Bangunan
**Stack:** Golang (Fiber v3) · PostgreSQL · Redis · Asynq · gRPC (Payment/Xendit, Notification/Mailjet) · RajaOngkir

---

## 1. Daftar Module Lengkap

### Core Modules (Monolith)

| # | Module | Deskripsi | Role Utama |
|---|--------|-----------|------------|
| 1 | **Auth** | Register, login, refresh token (JWT), verifikasi email (OTP), reset password, logout (blacklist token di Redis) | Semua |
| 2 | **User** | Profil, ganti password, manajemen alamat (multi-address, alamat proyek/gudang), upload avatar | User |
| 3 | **Supplier** | Registrasi toko, pengajuan verifikasi (upload dokumen: NIB/KTP), profil toko, rekening bank untuk payout, jam operasional, lokasi origin pengiriman | Supplier |
| 4 | **Admin** | Approval supplier, manajemen kategori, manajemen user (ban/suspend), setting komisi platform, moderasi produk & review | Admin |
| 5 | **Category** | Kategori hierarkis (parent-child): Semen & Agregat → Semen Portland, Pasir, dst. Slug untuk SEO | Admin (kelola), publik (baca) |
| 6 | **Product** | CRUD produk, varian (ukuran/satuan: sak, batang, m³, dus), multi-gambar, spesifikasi teknis (JSON: merk, dimensi, berat), berat & volume per unit (krusial untuk ongkir), min. order quantity, status draft/aktif/nonaktif | Supplier |
| 7 | **Inventory** | Manajemen stok per varian, stock reservation saat checkout (Redis lock + DB), riwayat pergerakan stok, notifikasi stok menipis | Supplier |
| 8 | **Search & Catalog** | Full-text search (PostgreSQL `tsvector` + GIN index), filter (kategori, harga, lokasi supplier, rating, merk), sorting, pagination cursor-based | Publik |
| 9 | **Cart** | Keranjang per user (persist di DB, cache di Redis), grouping item per supplier, validasi stok & min. order saat load | User |
| 10 | **Checkout** | Ringkasan order, split order otomatis per supplier, kalkulasi ongkir (RajaOngkir: berat + volumetrik), pemilihan kurir, validasi ulang harga & stok, pembuatan order + reservasi stok (transaksional) | User |
| 11 | **Order** | State machine status order (pending_payment → paid → processed → shipped → delivered → completed / cancelled / refunded), riwayat status, pembatalan, konfirmasi penerimaan, auto-complete setelah N hari | User, Supplier, Admin |
| 12 | **Shipping** | Integrasi RajaOngkir (cost & waybill tracking), input resi oleh supplier, opsi **pengiriman armada supplier sendiri** (flat rate/per km — umum untuk material berat seperti pasir & besi), zona jangkauan pengiriman supplier | Supplier |
| 13 | **Review & Rating** | Review per produk (hanya pembeli yang order-nya completed), rating agregat di produk & toko, balasan supplier, foto review | User |
| 14 | **Wishlist** | Simpan produk favorit | User |
| 15 | **Payout / Settlement** | Escrow flow: dana ditahan sampai order completed → saldo supplier → request withdraw → admin approve → disbursement (via payment service). Ledger transaksi supplier, potongan komisi platform | Supplier, Admin |
| 16 | **Notification (in-app)** | Notifikasi dalam aplikasi (order update, approval, payout) + trigger email via gRPC ke notification service. Read/unread status | Semua |
| 17 | **Report & Analytics** | Dashboard supplier (penjualan, produk terlaris), dashboard admin (GMV, komisi, supplier aktif), export CSV | Supplier, Admin |
| 18 | **Media / File Upload** | Upload gambar produk & dokumen verifikasi ke object storage (S3/MinIO), presigned URL, validasi tipe & ukuran, resize/compress via worker | Semua |
| 19 | **Audit Log** | Pencatatan aksi sensitif (approval, perubahan harga, ban user, payout) untuk keperluan audit admin | Admin |
| 20 | **Banner / Promo Display** | Banner homepage & featured products yang dikelola admin (bukan sistem voucher — voucher tetap ditunda sesuai keputusan sebelumnya) | Admin |

### External Services (gRPC — sudah ada di rancangan)

| Service | Tanggung Jawab |
|---------|----------------|
| **Payment Service (Xendit)** | Create invoice/VA/e-wallet, terima webhook Xendit, disbursement untuk payout supplier, refund |
| **Notification Service (Mailjet)** | Kirim email transaksional (verifikasi, order, payout) berbasis template |

### Background Jobs (Asynq Worker)

- `email:send` — delegasi kirim email via notification service
- `order:expire` — batalkan order yang tidak dibayar dalam X jam + release stok reservasi
- `order:autocomplete` — auto-complete order N hari setelah delivered
- `payment:sync` — rekonsiliasi status pembayaran ke payment service
- `stock:release` — release reservasi stok yang expired
- `media:process` — resize/compress gambar setelah upload
- `report:generate` — generate laporan/export CSV besar
- `notification:lowstock` — cek & kirim notifikasi stok menipis (scheduled/cron via Asynq Scheduler)

---

## 2. Struktur Direktori Konkret

```
marketplace-bahan-bangunan/
├── cmd/
│   ├── api/main.go                  # HTTP server (Fiber v3)
│   ├── worker/main.go               # Asynq worker + scheduler
│   └── migrate/main.go              # runner migrasi + seeder
│
├── config/
│   ├── config.go                    # struct config + Viper loader
│   └── config.yaml                  # default config (di-override .env)
│
├── internal/
│   ├── domain/                      # LAYER 1: entity + kontrak (pure Go, tanpa dependency luar)
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
│   │   ├── repository/              # interface repository per domain
│   │   │   ├── user_repository.go
│   │   │   ├── product_repository.go
│   │   │   ├── order_repository.go
│   │   │   └── ... (per domain)
│   │   ├── service/                 # interface external service
│   │   │   ├── payment_service.go   # kontrak ke payment gRPC
│   │   │   ├── notification_service.go
│   │   │   ├── shipping_gateway.go  # kontrak RajaOngkir
│   │   │   └── storage_service.go   # kontrak object storage
│   │   └── errs/
│   │       └── errors.go            # domain error types (ErrNotFound, ErrInsufficientStock, ...)
│   │
│   ├── usecase/                     # LAYER 2: business logic (flat — 1 file per module, package tunggal `usecase`)
│   │   ├── auth_usecase.go
│   │   ├── user_usecase.go
│   │   ├── supplier_usecase.go
│   │   ├── admin_usecase.go
│   │   ├── category_usecase.go
│   │   ├── product_usecase.go
│   │   ├── inventory_usecase.go
│   │   ├── catalog_usecase.go       # search, filter, product listing publik
│   │   ├── cart_usecase.go
│   │   ├── checkout_usecase.go      # orkestrasi: validasi → ongkir → split order → payment
│   │   ├── order_usecase.go
│   │   ├── shipping_usecase.go
│   │   ├── review_usecase.go
│   │   ├── wishlist_usecase.go
│   │   ├── payout_usecase.go
│   │   ├── notification_usecase.go
│   │   ├── report_usecase.go
│   │   └── media_usecase.go
│   │
│   ├── delivery/                    # LAYER 3: transport
│   │   └── http/
│   │       ├── handler/             # 1 file per module, mapping ke usecase
│   │       ├── middleware/
│   │       │   ├── auth.go          # JWT parse + inject claims ke context
│   │       │   ├── rbac.go          # RequireRole("admin"|"supplier"|"user")
│   │       │   ├── ratelimit.go     # sliding window via Redis
│   │       │   ├── recover.go
│   │       │   ├── requestid.go
│   │       │   └── logger.go        # structured request log
│   │       ├── dto/                 # request/response struct + tag validasi per module
│   │       └── route/
│   │           ├── route.go         # registrasi utama
│   │           ├── public.go        # /api/v1/catalog, /auth
│   │           ├── user.go          # /api/v1/user/* (JWT + role user)
│   │           ├── supplier.go      # /api/v1/supplier/*
│   │           └── admin.go         # /api/v1/admin/*
│   │
│   ├── repository/                  # LAYER 4: implementasi persistence
│   │   ├── postgres/                # pgx + sqlc (atau GORM sesuai preferensi tim)
│   │   │   └── ... (per domain)
│   │   └── redis/
│   │       ├── cache.go             # cache produk/kategori
│   │       ├── session.go
│   │       └── stock_lock.go        # distributed lock reservasi stok
│   │
│   ├── infra/                       # adapter dunia luar
│   │   ├── database/postgres.go     # pool + health
│   │   ├── cache/client.go          # inisialisasi & koneksi Redis client (nama `cache` agar tidak duplikat dengan repository/redis)
│   │   ├── grpcclient/
│   │   │   ├── payment.go           # implementasi domain/service/payment_service.go
│   │   │   └── notification.go
│   │   ├── rajaongkir/client.go
│   │   ├── storage/s3.go            # S3/MinIO
│   │   └── queue/
│   │       ├── client.go            # Asynq client (enqueue)
│   │       ├── tasks.go             # definisi task type + payload
│   │       └── handlers/            # handler per task (dipakai cmd/worker)
│   │
│   └── app/
│       ├── server.go                # wiring Fiber: middleware global, route, graceful shutdown
│       └── container.go             # dependency injection manual (atau wire/fx)
│
├── pkg/                             # utilitas reusable, tidak tahu domain
│   ├── jwt/
│   ├── response/                    # envelope standar {success, message, data, meta, errors}
│   ├── pagination/                  # cursor & offset helper
│   ├── validator/                   # go-playground/validator + custom rules (phone ID, dll)
│   ├── hash/                        # bcrypt/argon2
│   ├── logger/                      # zerolog/zap setup
│   └── apperr/                      # mapping domain error → HTTP status
│
├── proto/                           # .proto payment & notification + generated code
│   ├── payment/v1/payment.proto
│   └── notification/v1/notification.proto
│
├── migrations/                      # golang-migrate: 000001_create_users.up.sql / .down.sql
├── seeds/                           # data awal: roles, kategori bahan bangunan, admin default
├── docs/
│   ├── openapi.yaml                 # Swagger/OpenAPI spec
│   └── erd.md                       # ERD + diagram Mermaid dari planning sebelumnya
├── deploy/
│   ├── docker-compose.yml           # postgres, redis, minio, api, worker
│   ├── Dockerfile.api
│   └── Dockerfile.worker
├── scripts/                         # helper: gen-proto.sh, lint.sh
├── test/
│   ├── integration/                 # test dengan testcontainers
│   └── mocks/                       # generated mocks (mockery)
├── .env.example
├── .golangci.yml
├── Makefile                         # run, migrate, seed, gen-proto, test, lint
└── go.mod
```

---

## 3. Keputusan Desain Penting

**Split order per supplier.** Satu checkout bisa berisi produk dari beberapa supplier → dipecah menjadi beberapa `orders` (satu per supplier) di bawah satu `checkout_group_id`, tapi **satu invoice pembayaran** ke Xendit. Ongkir dihitung per order (origin supplier berbeda-beda).

**Berat & volumetrik wajib di produk.** Bahan bangunan berat (semen 40kg/sak, besi panjang). Ongkir = max(berat aktual, berat volumetrik). Untuk item yang tidak masuk akal dikirim ekspedisi (pasir per m³), aktifkan opsi *pengiriman armada supplier* dengan zona jangkauan.

**Stock reservation, bukan langsung potong.** Saat order dibuat: stok di-reserve (row `stock_reservations` + Redis lock untuk race condition). Dibayar → reservasi dikonversi jadi pengurangan stok. Expired → job `order:expire` release reservasi.

**Escrow & ledger untuk payout.** Dana masuk platform dulu. Order completed → kredit ke saldo supplier (dicatat di `ledger_entries`, komisi platform didebit). Withdraw → approval admin → disbursement via payment service. Ledger append-only supaya saldo selalu bisa diaudit.

**Order status sebagai state machine.** Transisi valid didefinisikan eksplisit di usecase (mis. `paid → processed` hanya oleh supplier, `cancelled` hanya dari `pending_payment`/`paid` dengan refund). Setiap transisi dicatat di `order_status_histories`.

**RBAC via route group + middleware.** Karena `role_id` sudah FK langsung di `users`, cukup: `admin := v1.Group("/admin", mw.Auth(), mw.RequireRole("admin"))`. Tidak perlu permission table dulu — bisa dievolusi nanti kalau butuh granular permission.

---

## 4. Urutan Fase Implementasi yang Disarankan

1. **Fondasi** — skeleton project, config, database + migrasi awal, Redis, logger, response envelope, error handling, middleware dasar, Makefile, docker-compose.
2. **Auth + RBAC** — register/login/refresh, verifikasi email (lewat worker + notification service stub), middleware role.
3. **Supplier onboarding** — registrasi toko, upload dokumen (media module), approval admin.
4. **Katalog** — category, product + varian + gambar, inventory, search & filter publik.
5. **Transaksi inti** — cart → checkout (ongkir RajaOngkir, split order) → integrasi payment service → order state machine → shipping/resi.
6. **Pasca-transaksi** — review, notification in-app, payout/ledger + withdraw.
7. **Pelengkap** — wishlist, report/analytics, audit log, banner, hardening (rate limit, test integrasi, load test ringan).

Setiap fase menghasilkan sesuatu yang bisa di-demo, dan dependensi antar fase mengalir satu arah.

---

## 5. Catatan Fiber v3

- Fiber v3 masih tahap release candidate — API bisa berubah minor sampai stable. Pin versi di `go.mod` dan baca changelog tiap upgrade.
- Perubahan utama dari v2: `fiber.Ctx` menjadi interface, binding via `c.Bind().Body(&dto)`, middleware bawaan direstrukturisasi.
- Karena semua kode Fiber terkunci di `internal/delivery/http`, risiko breaking change terisolasi di satu layer — usecase dan repository tidak tersentuh.
