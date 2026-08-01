# CLAUDE.md — Marketplace Bahan Bangunan ("Bangunin" / [nama final TBD])

Dokumen ini adalah konteks utama project. Baca dan patuhi seluruh keputusan di sini sebelum menulis kode apa pun. Jangan mengubah keputusan arsitektur tanpa konfirmasi eksplisit dari saya.

## 1. Deskripsi Project

Marketplace vertikal khusus bahan bangunan & material konstruksi Indonesia. Supplier (distributor/toko bangunan) menjual ke kontraktor, pemborong, dan pemilik proyek. Karakteristik unik: AOV tinggi (Rp1–5 jt), material berat (ongkir berbasis berat aktual vs volumetrik), satuan jual khas (sak, batang, m³, dus), pembayaran escrow, split order per supplier.

## 2. Tech Stack (FINAL — jangan diganti)

- **Go 1.23+**, HTTP framework **Fiber v3** (pin versi di go.mod; v3 masih RC — isolasi semua kode Fiber di `internal/delivery/http`)
- **PostgreSQL 16** (pgx + sqlc), **Redis 7** (cache, session, rate limit, stock lock, backend Asynq)
- **Asynq** untuk background jobs + scheduler
- **gRPC** ke 2 service eksternal terpisah: payment-service (Xendit) & notification-service (Mailjet). Kontrak di `proto/`
- **RajaOngkir (Komerce)** via HTTP client untuk ongkir & tracking
- Object storage: **Cloudflare R2** (S3-compatible; dev pakai MinIO)
- zerolog, Viper, golang-migrate, go-playground/validator, golang-jwt/jwt v5, testify + mockery + testcontainers-go, golangci-lint
- Docker + docker-compose untuk lokal (postgres, redis, minio, api, worker)

## 3. Arsitektur & Struktur Direktori

Clean Architecture, dependency mengalir ke dalam: `delivery → usecase → domain ← repository/infra`.

```
cmd/{api,worker,migrate}/main.go
config/
internal/
  domain/{entity, repository(interface), service(interface eksternal), errs}
  usecase/{auth,user,supplier,admin,category,product,inventory,catalog,cart,checkout,order,shipping,review,wishlist,payout,notification,report,media}
  delivery/http/{handler, middleware, dto, route}
  repository/{postgres, redis}
  infra/{database, redis, grpcclient, rajaongkir, storage, queue}
  app/{server.go, container.go}   # DI manual
pkg/{jwt, response, pagination, validator, hash, logger, apperr}
proto/  migrations/  seeds/  docs/  deploy/  scripts/  test/
```

Aturan:
- Semua interface repo & service eksternal didefinisikan di `internal/domain`, implementasi di `repository/` atau `infra/`.
- Handler TIDAK berisi business logic — hanya bind DTO, panggil usecase, map error → response.
- File output user: `/mnt` tidak relevan — ini repo lokal biasa.

## 4. Domain & Skema Data (ringkasan — detail di docs/erd.md)

Tabel inti (±28): roles, users (role_id FK langsung — TANPA tabel pivot), user_addresses, suppliers (1:1 users, status pending/approved/rejected/suspended, origin_city_id, own_fleet_enabled), supplier_documents, supplier_bank_accounts, categories (hierarkis parent_id), products (specs jsonb, search_vector tsvector+GIN, rating denormalized), product_variants (price, weight_gram WAJIB, dimensi, unit, min_order_qty, stock), product_images, stock_movements (append-only), stock_reservations (active/converted/released, expires_at), carts, cart_items, checkout_groups, payments (1:1 checkout_group, xendit_invoice_id), orders (split per supplier di bawah checkout_group, snapshot shipping_address jsonb, commission_amount), order_items (snapshot nama/harga/berat), order_status_histories, shipments (method courier|supplier_fleet), reviews (1:1 order_item), supplier_balances, ledger_entries (append-only, balance_after), withdraw_requests, notifications, wishlists, audit_logs, banners.

ID: UUID v7. Timestamps: timestamptz. Soft delete hanya jika perlu (default: tidak).

## 5. Keputusan Desain Kritis (WAJIB diikuti)

1. **Split order per supplier**: 1 checkout → N orders (per supplier) di bawah 1 checkout_group → 1 invoice Xendit. Ongkir dihitung per order.
2. **Ongkir**: chargeable weight = max(berat aktual, volumetrik (p×l×t/6000)). Dukung 2 metode: kurir RajaOngkir & armada supplier (flat/zona km).
3. **Stock reservation**: checkout confirm → Redis lock per variant → validasi → insert reservation (expires 2 jam). Dibayar → convert ke stock_movements + kurangi stok. Expired → job release. JANGAN pernah langsung memotong stok saat checkout.
4. **Escrow + ledger**: dana masuk platform. Order completed → ledger credit_order + debit_commission (komisi 4%) → saldo supplier. Withdraw → approval admin → disbursement via payment-service. `ledger_entries` append-only, tidak boleh ada UPDATE/DELETE.
5. **Order state machine**: pending_payment → paid → processed → shipped → delivered → completed; cabang expired/cancelled(+refund). Transisi divalidasi di usecase per aktor (supplier/user/admin/system), semua dicatat di order_status_histories.
6. **RBAC**: middleware `Auth()` + `RequireRole(role)`; route group `/api/v1/{public|user|supplier|admin}`.
7. **Response envelope standar** (pkg/response): `{success, message, data, meta, errors}`. Error domain di-map ke HTTP status via pkg/apperr.
8. **Snapshot** harga/nama/alamat di order_items & orders — jangan join ke data master untuk data historis order.

## 6. Background Jobs (Asynq)

`email:send`, `order:expire` (batalkan + release reservasi), `order:autocomplete` (N hari setelah delivered), `payment:sync`, `stock:release`, `media:process` (resize gambar), `report:generate`, `notification:lowstock` (cron).

## 7. Konvensi Kode

- Bahasa kode/komentar/commit: English. Pesan error ke user (response message): Bahasa Indonesia.
- Nama file snake_case, package singular pendek.
- Setiap usecase punya unit test (mock repo via mockery). Repository ditest dengan testcontainers.
- Migrasi: satu perubahan = satu pasang file up/down. Jangan edit migrasi yang sudah dibuat — buat migrasi baru.
- Validasi DTO di layer delivery (validator tags), validasi bisnis di usecase.
- Jangan commit secret. Semua konfigurasi via env (lihat .env.example).
- Setelah perubahan signifikan: `make lint && make test` harus hijau sebelum lanjut.

## 8. Roadmap Fase (kerjakan berurutan, satu fase = satu rangkaian PR)

1. **Fondasi**: skeleton sesuai struktur di atas, config, koneksi PG+Redis, logger, response envelope, apperr, middleware dasar (recover, requestid, logger, ratelimit), migrasi awal (roles, users), seeds roles+admin, docker-compose, Makefile (run, migrate, seed, test, lint, gen-proto), health endpoint.
2. **Auth + RBAC**: register/login/refresh/logout, verifikasi email OTP (job email:send → stub notification-service dulu), reset password, middleware Auth+RequireRole.
3. **Supplier onboarding**: registrasi toko, upload dokumen (media module + MinIO), approval/reject admin + audit_logs.
4. **Katalog**: category CRUD (admin), product+variant+image CRUD (supplier), inventory & movements, catalog publik (search tsvector, filter, pagination cursor).
5. **Transaksi inti**: cart → checkout preview (ongkir RajaOngkir + armada) → confirm (lock+reservation+split order) → payment-service gRPC (implement service beneran di repo terpisah/folder services/) → webhook paid → order state machine → shipment.
6. **Pasca-transaksi**: review, notifikasi in-app, payout (ledger, withdraw, approval, disbursement).
7. **Pelengkap**: wishlist, report, banner, hardening + integration test end-to-end.

## 9. Cara Kerja yang Saya Harapkan dari Claude Code

- Sebelum mulai fase: tampilkan rencana file yang akan dibuat/diubah, tunggu konfirmasi saya untuk fase besar.
- Kerjakan dalam potongan yang bisa dikompilasi & ditest — jangan tulis 50 file sekaligus tanpa pernah menjalankan build.
- Jika ada ambiguitas terhadap dokumen ini, tanya dulu; jangan mengarang keputusan arsitektur baru.
- Update file PROGRESS.md setiap menyelesaikan sub-task (checklist per fase).
