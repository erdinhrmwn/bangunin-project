# PROGRESS — Marketplace Bahan Bangunan

Checklist FR & AC per fase. Centang hanya setelah lolos verifikasi (build/test/manual). Lihat docs/requirements.md untuk detail lengkap tiap item.

---

## FASE 1 — Fondasi

### Functional Requirements
- [ ] FR-1.1 Struktur project sesuai CLAUDE.md §3
- [ ] FR-1.2 Config (Viper + env override, fail-fast)
- [ ] FR-1.3 Database (pgxpool + golang-migrate)
- [ ] FR-1.4 Migrasi awal (roles, users)
- [ ] FR-1.5 Seeder idempotent (roles + admin)
- [ ] FR-1.6 Redis client + helper TTL
- [ ] FR-1.7 Logger (zerolog JSON + request log middleware)
- [ ] FR-1.8 Middleware dasar (recover, request ID, CORS, rate limit)
- [ ] FR-1.9 Response & error (pkg/response, pkg/apperr)
- [ ] FR-1.10 Health endpoint
- [ ] FR-1.11 Worker skeleton (Asynq + system:heartbeat)
- [ ] FR-1.12 Tooling (Makefile, golangci-lint, .env.example, docker-compose)
- [ ] FR-1.13 Graceful shutdown

### Acceptance Criteria
- [ ] AC-1.a docker compose up sehat + migrate + seed + health 200
- [ ] AC-1.b Redis mati → 503; panic → 500 rapi, proses hidup
- [ ] AC-1.c lint & test hijau; migrate down/up bersih; seed idempotent
- [ ] AC-1.d Rate limit 429 pada 70 req/menit

---

## FASE 2 — Auth + RBAC

### Functional Requirements
- [ ] FR-2.1 Register
- [ ] FR-2.2 Verifikasi email (OTP)
- [ ] FR-2.3 Login (JWT + refresh + brute force guard)
- [ ] FR-2.4 Refresh (rotasi)
- [ ] FR-2.5 Logout (blacklist jti)
- [ ] FR-2.6 Reset password
- [ ] FR-2.7 Middleware Auth
- [ ] FR-2.8 Middleware RequireRole
- [ ] FR-2.9 Profil dasar (/user/me)
- [ ] FR-2.10 Notification stub (log-only)

### Acceptance Criteria
- [ ] AC-2.a Happy path register→verify→login→me→refresh→logout→token lama ditolak
- [ ] AC-2.b RBAC 403/401 sesuai role
- [ ] AC-2.c Refresh token reuse ditolak; reset password invalidasi sesi lama
- [ ] AC-2.d Unit test coverage kasus auth

---

## FASE 3 — Supplier Onboarding

### Functional Requirements
- [ ] FR-3.1 Media module (upload + resize)
- [ ] FR-3.2 Profil toko
- [ ] FR-3.3 Dokumen KYC
- [ ] FR-3.4 Rekening bank
- [ ] FR-3.5 Submit pengajuan
- [ ] FR-3.6 Review admin (approve/reject/suspend)
- [ ] FR-3.7 Audit log
- [ ] FR-3.8 Notifikasi
- [ ] FR-3.9 Guard RequireApprovedSupplier

### Acceptance Criteria
- [ ] AC-3.a Alur penuh profil→dokumen→submit→reject→perbaiki→approve
- [ ] AC-3.b Validasi file (tipe/ukuran) + resize worker
- [ ] AC-3.c Submit tanpa NIB → 422; endpoint berjualan sebelum approved → 403
- [ ] AC-3.d Migrasi baru up/down bersih

---

## FASE 4 — Katalog

### Functional Requirements
- [ ] FR-4.1 Kategori (admin CRUD + publik tree + cache)
- [ ] FR-4.2 Produk (supplier CRUD)
- [ ] FR-4.3 Varian
- [ ] FR-4.4 Gambar produk
- [ ] FR-4.5 Stok & movements
- [ ] FR-4.6 Katalog publik (search + filter + cursor)
- [ ] FR-4.7 Detail publik (cache invalidate)
- [ ] FR-4.8 search_vector trigger + GIN index
- [ ] FR-4.9 Toko publik
- [ ] FR-4.10 Low-stock job

### Acceptance Criteria
- [ ] AC-4.a Produk publish muncul & ditemukan di search
- [ ] AC-4.b Validasi publish (gambar, weight_gram, SKU unik)
- [ ] AC-4.c Stock adjustment (409 saat negatif, movements benar)
- [ ] AC-4.d Supplier suspended tidak muncul; cache invalidate cepat
- [ ] AC-4.e Integration test search dengan testcontainers

---

## FASE 5 — Transaksi Inti

### Functional Requirements
- [ ] FR-5.1 Cart
- [ ] FR-5.2 RajaOngkir client (ShippingGateway + mock mode)
- [ ] FR-5.3 Checkout preview
- [ ] FR-5.4 Checkout confirm (lock + TX + reservation)
- [ ] FR-5.5 Payment-service (services/payment, gRPC, Xendit)
- [ ] FR-5.6 Callback paid (idempotent)
- [ ] FR-5.7 Job order:expire
- [ ] FR-5.8 State machine order
- [ ] FR-5.9 Shipment
- [ ] FR-5.10 Listing order (user/supplier/admin)
- [ ] FR-5.11 Job order:autocomplete

### Acceptance Criteria
- [ ] AC-5.a E2E multi-supplier checkout→paid→ship→delivered→completed
- [ ] AC-5.b Concurrent confirm race → tepat satu berhasil
- [ ] AC-5.c Job expire release reservasi; callback paid duplikat idempotent
- [ ] AC-5.d Cancel saat paid → refund + stok kembali; transisi ilegal 409
- [ ] AC-5.e Payment-service proses terpisah; monolith degraded bila mati

---

## FASE 6 — Pasca-transaksi

### Functional Requirements
- [ ] FR-6.1 Settlement on complete (ledger + balance)
- [ ] FR-6.2 Ledger (append-only)
- [ ] FR-6.3 Withdraw
- [ ] FR-6.4 Approval & disbursement
- [ ] FR-6.5 Review
- [ ] FR-6.6 Notifikasi lengkap (notification-service asli)
- [ ] FR-6.7 Preferensi notifikasi ringan

### Acceptance Criteria
- [ ] AC-6.a Settlement ledger benar; idempotent
- [ ] AC-6.b Withdraw lifecycle + Σ ledger = balance; UPDATE ledger ditolak DB
- [ ] AC-6.c Review guard (403/409) + rating agregat benar
- [ ] AC-6.d Notification-service mati → tidak gagalkan transaksi (retry)

---

## FASE 7 — Pelengkap & Hardening

### Functional Requirements
- [ ] FR-7.1 Wishlist
- [ ] FR-7.2 Banner
- [ ] FR-7.3 Report supplier
- [ ] FR-7.4 Report admin
- [ ] FR-7.5 Hardening keamanan
- [ ] FR-7.6 Observabilitas (metrics + Grafana)
- [ ] FR-7.7 Kualitas (coverage, CI, seed-demo)
- [ ] FR-7.8 Dokumentasi (OpenAPI, README, PROGRESS.md final)

### Acceptance Criteria
- [ ] AC-7.a CI hijau + seed-demo + 3 role demo
- [ ] AC-7.b Metrics + Grafana + load test p95 < 300ms
- [ ] AC-7.c govulncheck bersih; callback internal tanpa secret → 401
- [ ] AC-7.d OpenAPI valid & 100% route coverage

---

## Keputusan

Catatan keputusan saat menemui ambiguitas (diisi berjalannya waktu, per fase).
