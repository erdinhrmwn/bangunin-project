# Requirements per Fase — Marketplace Bahan Bangunan

Dokumen acuan implementasi. Setiap fase berisi: tujuan, ruang lingkup, functional requirements (FR), endpoint, migrasi, background jobs, dan acceptance criteria (AC) yang harus lolos sebelum fase dinyatakan selesai. Simpan sebagai `docs/requirements.md` di repo — CLAUDE.md merujuk ke dokumen ini.

Konvensi: `FR-x.y` = functional requirement fase x nomor y. Semua endpoint di bawah prefix `/api/v1`. Semua response memakai envelope standar `{success, message, data, meta, errors}`.

---

## FASE 1 — Fondasi

**Tujuan:** repo bisa di-clone, `docker compose up`, migrate, seed, dan menyajikan health endpoint dengan seluruh tulang punggung (config, DB, Redis, logger, middleware, error handling) siap dipakai fase berikutnya.

**In scope:** skeleton, tooling, infrastruktur lokal, migrasi awal, middleware dasar.
**Out of scope:** semua fitur bisnis, gRPC client, RajaOngkir, storage.

### Functional Requirements
- **FR-1.1 Struktur project** sesuai CLAUDE.md §3. Setiap folder berisi minimal `doc.go` atau file nyata — tidak ada folder kosong yang tidak terpakai.
- **FR-1.2 Config**: Viper membaca `config.yaml` + override env (prefix `APP_`). Struct config typed: App{Name, Env, Port}, DB{DSN, MaxConns...}, Redis, JWT{Secret, AccessTTL, RefreshTTL}, dan section kosong untuk Xendit/RajaOngkir/R2 (diisi fase terkait). Aplikasi fail-fast dengan pesan jelas jika config wajib kosong.
- **FR-1.3 Database**: koneksi pgxpool dengan ping saat startup, graceful close saat shutdown. Migrasi via golang-migrate dijalankan `cmd/migrate` dan target Makefile.
- **FR-1.4 Migrasi awal**: `roles` (seed: admin, supplier, user) dan `users` sesuai ERD (uuid v7 default, role_id FK, unique email, index email & status). Up dan down keduanya berfungsi.
- **FR-1.5 Seeder**: idempotent (aman dijalankan berulang). Seed roles + 1 akun admin default dari env (`SEED_ADMIN_EMAIL`, `SEED_ADMIN_PASSWORD`).
- **FR-1.6 Redis**: client dengan ping saat startup; helper untuk get/set/del dengan TTL di `internal/repository/redis`.
- **FR-1.7 Logger**: zerolog JSON ke stdout; level dari config; field standar (service, env). Request log middleware mencatat method, path, status, latency, request_id.
- **FR-1.8 Middleware dasar**: recover (panic → 500 envelope + log stack), request ID (header `X-Request-ID`, generate jika kosong), CORS configurable, rate limit Redis sliding window (default 60 req/menit/IP, path-exempt: /health).
- **FR-1.9 Response & error**: `pkg/response` (Success, Error, paginated meta) dan `pkg/apperr` — tipe AppError{Code, Message, HTTPStatus} + mapper dari domain errs (`ErrNotFound`→404, `ErrValidation`→422, `ErrUnauthorized`→401, `ErrForbidden`→403, `ErrConflict`→409, default 500 tanpa bocor detail internal).
- **FR-1.10 Health**: `GET /health` → status app, db (ping), redis (ping), versi, uptime. 503 jika dependensi mati.
- **FR-1.11 Worker skeleton**: `cmd/worker` menjalankan Asynq server terhubung Redis dengan 1 task dummy `system:heartbeat` terdaftar; scheduler siap namun kosong.
- **FR-1.12 Tooling**: Makefile (run, run-worker, build, migrate-up/down/create, seed, test, lint, gen-proto placeholder), `.golangci.yml`, `.env.example` lengkap, docker-compose (postgres 16, redis 7, minio, api, worker) dengan healthcheck dan volume persist.
- **FR-1.13 Graceful shutdown**: SIGINT/SIGTERM → berhenti terima request, tunggu in-flight max 10 dtk, tutup DB/Redis.

### Acceptance Criteria
- AC-1.a `docker compose up -d` → semua container healthy; `make migrate-up && make seed` sukses; `curl /health` = 200 dengan status db & redis "ok".
- AC-1.b Matikan Redis → /health = 503; panic buatan di handler uji → 500 envelope rapi + stack di log, proses tidak mati.
- AC-1.c `make lint` dan `make test` hijau; migrate down lalu up kembali sukses; seed dijalankan 2x tanpa error/duplikat.
- AC-1.d Kirim 70 request/menit ke endpoint uji → sebagian ditolak 429 dengan envelope standar.

---

## FASE 2 — Auth + RBAC

**Tujuan:** siklus akun penuh: daftar, verifikasi email, login/refresh/logout, reset password, dan proteksi route per role.

**In scope:** auth user & supplier-account, JWT, OTP email (via stub notification), middleware RBAC.
**Out of scope:** profil supplier/toko (fase 3), OAuth/social login.

### Functional Requirements
- **FR-2.1 Register** `POST /auth/register` {name, email, password, phone, role: user|supplier}. Password min 8 char dengan angka; email unik (409 jika sudah ada); role admin TIDAK bisa didaftarkan lewat endpoint ini. Password di-hash bcrypt cost 12. Status akun `active`, `email_verified_at` null.
- **FR-2.2 Verifikasi email**: setelah register, generate OTP 6 digit, simpan di Redis `otp:verify:{userID}` TTL 10 menit, enqueue `email:send`. `POST /auth/verify-email` {email, otp} → set email_verified_at. `POST /auth/resend-otp` rate-limited 1x/menit per email. OTP salah 5x → OTP hangus.
- **FR-2.3 Login** `POST /auth/login` {email, password} → access token (JWT, TTL 15 mnt, claims: sub, role, jti) + refresh token (random opaque, TTL 7 hari, disimpan Redis `refresh:{token}`→userID). Email belum terverifikasi → 403 dengan kode khusus `EMAIL_NOT_VERIFIED`. Akun suspended/banned → 403. Brute force guard: 10 gagal/15 menit per email+IP → blokir sementara.
- **FR-2.4 Refresh** `POST /auth/refresh` {refresh_token} → rotasi: token lama dihapus, terbit pasangan baru. Token tidak dikenal → 401.
- **FR-2.5 Logout** `POST /auth/logout` (auth) → hapus refresh token + blacklist jti access token di Redis hingga expiry.
- **FR-2.6 Reset password**: `POST /auth/forgot-password` (selalu 200 agar tidak bocorkan keberadaan email; kirim OTP jika ada) → `POST /auth/reset-password` {email, otp, new_password} → invalidasi semua refresh token user.
- **FR-2.7 Middleware Auth**: parse Bearer JWT, tolak expired/blacklisted/invalid (401), inject claims ke context (`ctxutil.UserID(c)`, `ctxutil.Role(c)`).
- **FR-2.8 Middleware RequireRole(roles...)**: 403 jika role tidak sesuai. Route group `/user`, `/supplier`, `/admin` terpasang dan dibuktikan dengan endpoint `GET {group}/me`.
- **FR-2.9 Profil dasar**: `GET /user/me`, `PATCH /user/me` (name, phone), `PATCH /user/me/password` (verifikasi password lama).
- **FR-2.10 Notification stub**: implementasi `domain/service.NotificationService` versi log-only (cetak isi email ke log) di `infra/grpcclient` — dipakai sampai service asli tersedia; ganti implementasi tanpa mengubah usecase.

### Acceptance Criteria
- AC-2.a Alur happy path register→OTP (dibaca dari log stub)→verify→login→akses `/user/me`→refresh→logout→access token lama ditolak 401.
- AC-2.b User role `user` mengakses `/supplier/me` → 403; tanpa token → 401; token expired → 401.
- AC-2.c Refresh token yang sudah dirotasi dipakai ulang → 401. Reset password membuat semua sesi lama tidak berlaku.
- AC-2.d Unit test usecase auth coverage kasus: email duplikat, OTP salah/expired/5x gagal, brute force lock, role admin ditolak saat register.

---

## FASE 3 — Supplier Onboarding

**Tujuan:** akun supplier melengkapi profil toko, upload dokumen, dan disetujui/ditolak admin dengan jejak audit.

**In scope:** profil toko, dokumen KYC, rekening bank, media upload (MinIO/R2), approval admin, audit log, notifikasi status.
**Out of scope:** produk (fase 4).

### Functional Requirements
- **FR-3.1 Media module**: `POST /media/upload` (auth; multipart) → validasi tipe (jpg/png/webp/pdf) & ukuran (gambar ≤5MB, pdf ≤10MB) → simpan ke bucket dengan key `{scope}/{uuid}.{ext}` → kembalikan url + key. Enqueue `media:process` untuk resize gambar (max 1600px) — worker mengganti file asli. Implementasi `domain/service.StorageService` berbasis S3 API (MinIO lokal, R2 produksi).
- **FR-3.2 Profil toko**: `POST /supplier/profile` & `PUT /supplier/profile` {store_name, description, origin_city_id, pickup_address, own_fleet_enabled, fleet_coverage_km?, fleet_flat_rate?}. Slug digenerate unik dari store_name. origin_city_id wajib valid (untuk fase ini cukup integer > 0; validasi RajaOngkir menyusul fase 5). Satu user maksimal satu supplier (409).
- **FR-3.3 Dokumen**: `POST /supplier/documents` {doc_type: nib|ktp|npwp, file_key} — maksimal 1 dokumen aktif per tipe (upload ulang menggantikan, status kembali pending). `GET /supplier/documents`.
- **FR-3.4 Rekening bank**: CRUD `/supplier/bank-accounts`; tepat satu is_default; nomor rekening 6–20 digit.
- **FR-3.5 Submit pengajuan**: `POST /supplier/submit` — hanya boleh jika profil lengkap + minimal dokumen KTP & NIB ada. Status supplier: `draft → pending`. Tidak bisa submit saat sudah pending/approved.
- **FR-3.6 Review admin**: `GET /admin/suppliers?status=&q=&page=` (list + filter), `GET /admin/suppliers/:id` (detail + dokumen + presigned url), `POST /admin/suppliers/:id/approve` (set approved + verified_at), `POST /admin/suppliers/:id/reject` {reason} (wajib alasan; supplier bisa perbaiki dan submit ulang), `POST /admin/suppliers/:id/suspend` {reason}.
- **FR-3.7 Audit log**: setiap approve/reject/suspend tercatat di audit_logs {actor_id, action, entity, metadata: {reason, before, after}, ip}. `GET /admin/audit-logs` dengan filter action & date range.
- **FR-3.8 Notifikasi**: perubahan status supplier → notifikasi in-app (tabel notifications) + enqueue email. `GET /user/notifications` + `POST /user/notifications/:id/read` + unread count.
- **FR-3.9 Guard**: seluruh endpoint `/supplier/*` di fase berikutnya yang bersifat berjualan wajib melewati middleware `RequireApprovedSupplier` (403 `SUPPLIER_NOT_APPROVED` jika belum approved) — middleware dibuat di fase ini.

### Acceptance Criteria
- AC-3.a Alur penuh: supplier lengkapi profil → upload 2 dokumen → submit → admin reject dengan alasan → supplier menerima notifikasi + alasan → perbaiki → submit ulang → approve → notifikasi diterima, verified_at terisi, audit_logs berisi 2 entri.
- AC-3.b Upload file .exe atau 20MB → 422. File gambar besar ter-resize oleh worker (verifikasi dimensi hasil).
- AC-3.c Submit tanpa NIB → 422 dengan daftar kekurangan. Akses endpoint berjualan (dummy) sebelum approved → 403.
- AC-3.d Migrasi baru: suppliers, supplier_documents, supplier_bank_accounts, notifications, audit_logs — up/down bersih.

---

## FASE 4 — Katalog

**Tujuan:** supplier approved mengelola produk lengkap; publik dapat mencari dan menelusuri katalog dengan cepat.

**In scope:** kategori, produk+varian+gambar, stok & movements, listing publik + full-text search, cache.
**Out of scope:** cart/checkout (fase 5), review (fase 6).

### Functional Requirements
- **FR-4.1 Kategori (admin)**: CRUD `/admin/categories` hierarkis (parent_id, max 3 level), slug unik, is_active, sort_order. Hapus hanya jika tanpa anak & tanpa produk (409). Publik: `GET /categories` (tree, cache Redis 10 menit, invalidate saat mutasi). Seed ±20 kategori bahan bangunan realistis (Semen & Agregat, Besi & Baja, Kayu, Cat, Keramik & Granit, Sanitasi, Atap, Listrik, Pipa, dst).
- **FR-4.2 Produk (supplier, approved-only)**: CRUD `/supplier/products`. Field: category_id, name, description, specs (jsonb bebas: merk, material, dimensi), status draft|active|inactive. Publish (draft→active) hanya jika ≥1 varian aktif dan ≥1 gambar. Slug unik global.
- **FR-4.3 Varian**: CRUD nested `/supplier/products/:id/variants`. Field wajib: name, sku (unik per supplier), unit (enum: sak|batang|m3|dus|pcs|lembar|kg|roll|set), price>0, **weight_gram>0**, dimensi cm (boleh 0 untuk non-volumetrik), min_order_qty≥1, stock≥0. Update harga tidak mengubah order lama (snapshot — ditegakkan fase 5).
- **FR-4.4 Gambar**: attach dari media upload, max 8 per produk, tepat satu is_primary, sortable.
- **FR-4.5 Stok**: `POST /supplier/variants/:id/stock-adjustment` {qty(+/-), note} → tulis stock_movements (type in/out/adjustment, stock_after) dalam transaksi bersama update stok; stok tidak boleh negatif (409). `GET /supplier/variants/:id/movements` (riwayat, paginated).
- **FR-4.6 Katalog publik**: `GET /products?q=&category=&min_price=&max_price=&city=&sort=(newest|price_asc|price_desc|rating)&cursor=` — hanya produk active dari supplier approved. q memakai tsvector (name+description+specs merk) websearch_to_tsquery, ranking ts_rank. Response ringkas: id, slug, name, primary_image, price_min–max, unit, supplier{name, city}, rating. Pagination cursor-based.
- **FR-4.7 Detail publik**: `GET /products/:slug` → lengkap varian aktif + stok tersedia (stok − reservasi aktif; sebelum fase 5, reservasi=0), gambar, specs, profil singkat supplier. Cache Redis 60 dtk, invalidate saat supplier mengubah produk.
- **FR-4.8 search_vector** dijaga trigger PostgreSQL (INSERT/UPDATE), index GIN. EXPLAIN memakai index.
- **FR-4.9 Toko publik**: `GET /suppliers/:slug` → profil + produknya (paginated).
- **FR-4.10 Low-stock job**: scheduler harian `notification:lowstock` — varian stok < ambang (default 5, per supplier configurable nanti; hardcode default dulu) → notifikasi supplier, maksimal 1x/hari/varian.

### Acceptance Criteria
- AC-4.a Supplier membuat produk "Semen Portland 40kg" 2 varian + 3 gambar → publish → muncul di `GET /products?q=semen` dan pencarian "portland 40" juga menemukannya.
- AC-4.b Publish tanpa gambar → 422. Varian tanpa weight_gram → 422. SKU duplikat dalam satu supplier → 409.
- AC-4.c Adjustment −10 pada stok 5 → 409; +10 lalu −3 → movements 2 baris dengan stock_after benar.
- AC-4.d Produk supplier suspended tidak muncul di katalog publik. Cache: ubah nama produk → detail publik berubah ≤ invalidasi (bukan menunggu 60 dtk).
- AC-4.e Integration test repository search dengan testcontainers: seeding 50 produk, query q+filter+sort mengembalikan hasil dan urutan benar.

---

## FASE 5 — Transaksi Inti

**Tujuan:** alur uang dan barang end-to-end: cart → checkout (ongkir) → bayar via payment-service (Xendit) → order berjalan di state machine → dikirim → diterima.

**In scope:** cart, checkout preview/confirm, RajaOngkir client, stock reservation + Redis lock, payment-service gRPC (dibangun sungguhan), webhook, order lifecycle, shipment, job expire.
**Out of scope:** review, payout (fase 6).

### Functional Requirements
- **FR-5.1 Cart**: `GET /user/cart`, `POST /user/cart/items` {variant_id, qty}, `PATCH .../items/:id`, `DELETE .../items/:id`. Merge qty untuk varian sama. Response digrup per supplier, tiap item membawa flag masalah: out_of_stock, below_min_order, price_changed(sejak ditambahkan), product_inactive. Item bermasalah tidak menghalangi melihat cart tetapi menghalangi checkout.
- **FR-5.2 RajaOngkir client** (`infra/rajaongkir`): implementasi `domain/service.ShippingGateway` — GetCost(originCityID, destCityID, weightGram, couriers[]) dan (fase ini stub) TrackWaybill. Timeout 5 dtk, retry 1x, error → `ErrShippingUnavailable`. Konfigurasi API key via env; sediakan mock mode (`RAJAONGKIR_MOCK=true`) yang mengembalikan tarif deterministik untuk dev/test.
- **FR-5.3 Checkout preview** `POST /user/checkout/preview` {address_id, items[]}: validasi semua item (stok, min order, aktif, supplier approved) → grup per supplier → per grup hitung chargeable weight = Σ max(weight, volumetrik p·l·t/6000) → panggil ongkir paralel per grup → kembalikan: grup[{supplier, items, subtotal, shipping_options[kurir/service/etd/cost + opsi armada supplier bila enabled & jarak dalam coverage]}], grand_total_estimate. Alamat harus milik user.
- **FR-5.4 Checkout confirm** `POST /user/checkout/confirm` {address_id, items[], shipping_choice per grup}: urutan atomik — (1) Redis lock `stock:{variantID}` (SETNX+TTL 10 dtk, sorted by ID untuk hindari deadlock), (2) TX DB: validasi ulang harga & stok efektif (stok − reservasi aktif), insert checkout_group, orders per supplier (order_number format `ORD-YYYYMMDD-XXXXX`, commission_amount = 4% subtotal, snapshot alamat jsonb), order_items snapshot, stock_reservations (expires_at = now+2 jam), order_status_histories awal, (3) commit, release lock, (4) panggil payment-service CreateInvoice(amount=grand total, external_id=group_number) → simpan payments, (5) response invoice_url. Kegagalan langkah 4 → tandai group `payment_failed` dan release reservasi (kompensasi), 502 ke user.
- **FR-5.5 Payment-service (repo/folder terpisah `services/payment`)**: gRPC server Go mandiri. RPC: CreateInvoice, GetInvoice, (CreateDisbursement disiapkan untuk fase 6). Integrasi Xendit Invoice API; terima webhook Xendit di endpoint HTTP-nya sendiri dengan verifikasi callback token; meneruskan event ke monolith via RPC `PaymentEventCallback` (monolith mengekspos gRPC server kecil ATAU payment-service memanggil webhook HTTP monolith `POST /internal/payments/callback` yang diamankan shared secret — pilih HTTP callback, lebih sederhana; keputusan final: HTTP callback + header `X-Internal-Secret`). Sediakan mock mode penuh tanpa Xendit untuk dev.
- **FR-5.6 Callback paid** (idempotent, kunci: xendit_invoice_id + status): payments.paid → TX: checkout_group paid, semua orders paid + history, konversi reservasi → stock_movements(out) + kurangi stok, enqueue email & notifikasi ke pembeli dan tiap supplier. Callback duplikat tidak menggandakan efek.
- **FR-5.7 Job order:expire**: scheduler tiap menit — checkout_group pending melewati payments.expires_at → group expired, orders expired + history(actor system), release reservasi, notifikasi pembeli. Race dengan pembayaran: transisi divalidasi state machine (paid tidak bisa menjadi expired).
- **FR-5.8 State machine order** (usecase order, tabel transisi eksplisit): pending_payment→{paid, expired, cancelled(user)}; paid→{processed(supplier), cancelled(user|supplier → trigger refund via payment-service + kembalikan stok)}; processed→shipped(supplier); shipped→delivered(supplier utk armada | job tracking utk kurir — fase ini: endpoint supplier menandai delivered); delivered→completed(user confirm | job autocomplete 3 hari). Transisi ilegal → 409 `INVALID_STATUS_TRANSITION`. Setiap transisi 1 baris history.
- **FR-5.9 Shipment**: `POST /supplier/orders/:id/ship` {method, courier_code?, tracking_number?} — kurir wajib resi; armada wajib supplier own_fleet_enabled. `GET /user/orders/:id` menampilkan shipment & timeline history.
- **FR-5.10 Listing order**: user `GET /user/orders?status=`, supplier `GET /supplier/orders?status=`, admin `GET /admin/orders` (+filter supplier/user/tanggal, dan `POST /admin/orders/:id/force-status` dengan audit log untuk intervensi).
- **FR-5.11 Job order:autocomplete**: delivered > 3 hari → completed (actor system). (Efek saldo supplier menyusul fase 6 — desain event: fungsi `onOrderCompleted` dipanggil dari satu titik.)

### Acceptance Criteria
- AC-5.a E2E (mock RajaOngkir + mock payment): 2 supplier dalam 1 cart → preview menampilkan 2 grup ongkir → confirm → 2 orders + 1 invoice → callback paid → stok berkurang benar, reservasi converted, notifikasi terkirim → supplier A ship (kurir), supplier B ship (armada) → delivered → user confirm completed di A; B completed via job (uji dengan memundurkan delivered_at).
- AC-5.b Dua request confirm bersamaan memperebutkan stok terakhir (test paralel) → tepat satu berhasil, satu 409, tanpa stok negatif.
- AC-5.c Tidak dibayar → job expire: reservasi released, stok efektif pulih, order expired. Callback paid dikirim 2x → efek tunggal.
- AC-5.d Cancel saat paid → refund dipanggil (mock tercatat), stok kembali, history lengkap. Transisi ilegal (pending→shipped) → 409.
- AC-5.e Payment-service berjalan sebagai proses terpisah di docker-compose; monolith tetap start (degraded, checkout 502) bila service mati — bukan crash.

---

## FASE 6 — Pasca-transaksi

**Tujuan:** kepercayaan & perputaran uang: review produk, notifikasi lengkap, dan payout supplier ber-ledger auditabel.

### Functional Requirements
- **FR-6.1 Settlement on complete**: `onOrderCompleted` → TX: ledger credit_order (subtotal + ongkir jika armada sendiri) dan debit_commission (4% subtotal), keduanya dengan balance_after; update supplier_balances (pending→available sesuai desain: sederhanakan — saldo masuk langsung available saat completed). Idempotent per order (unique constraint ref).
- **FR-6.2 Ledger**: append-only (tanpa endpoint update/delete; DB: revoke UPDATE/DELETE atau trigger blokir). `GET /supplier/ledger?type=&from=&to=` paginated + saldo berjalan. Konsistensi: Σ ledger = balance (dibuktikan test).
- **FR-6.3 Withdraw**: `POST /supplier/withdraws` {amount, bank_account_id} — amount ≥ Rp50.000, ≤ available; TX: hold (available−, pending_withdraw baris ledger TIDAK ditulis dulu — hold cukup di balances + status request). `GET /supplier/withdraws`. Batal oleh supplier hanya saat status requested.
- **FR-6.4 Approval & disbursement**: `POST /admin/withdraws/:id/approve` → panggil payment-service CreateDisbursement → status processing; callback disbursement completed → status disbursed + ledger debit_withdraw + lepaskan hold; failed → kembalikan hold, status failed + notifikasi. `POST /admin/withdraws/:id/reject` {reason} → kembalikan hold. Semua aksi admin → audit log.
- **FR-6.5 Review**: `POST /user/orders/:orderId/items/:itemId/review` {rating 1–5, comment?, image_keys[] ≤3} — hanya pemilik order berstatus completed, satu review per order_item (409). Balasan supplier: `POST /supplier/reviews/:id/reply` sekali. Agregat rating_avg & rating_count produk diupdate transaksional. Publik: `GET /products/:slug/reviews?rating=&cursor=`.
- **FR-6.6 Notifikasi lengkap**: pastikan semua event mengirim in-app + email: order paid/shipped/delivered/completed/cancelled/expired (pembeli & supplier sesuai konteks), review baru (supplier), balasan review (pembeli), withdraw lifecycle (supplier), low stock. Email via notification-service asli (`services/notification`, gRPC, template Mailjet, mock mode) menggantikan stub fase 2 — tanpa mengubah usecase.
- **FR-6.7 Preferensi ringan**: `PATCH /user/me/notification-settings` {email_marketing: bool} — email transaksional selalu terkirim.

### Acceptance Criteria
- AC-6.a Order Rp2.000.000 (ongkir kurir) completed → ledger: credit 2.000.000, debit 80.000; available = 1.920.000. Order kedua dengan armada sendiri → ongkir ikut dikredit. Jalankan `onOrderCompleted` 2x → efek tunggal.
- AC-6.b Withdraw 1.000.000 → available turun seketika; reject → kembali; approve+disbursement sukses (mock) → ledger debit_withdraw, Σ ledger = balance. Percobaan UPDATE baris ledger via SQL → ditolak DB.
- AC-6.c Review pada order belum completed → 403; review kedua item sama → 409; rating produk terhitung benar setelah 3 review (avg & count).
- AC-6.d Notification-service asli berjalan di compose; matikan → aksi tetap sukses, email masuk retry Asynq (bukan gagal transaksi).

---

## FASE 7 — Pelengkap & Hardening

**Tujuan:** fitur pendukung, laporan, dan kesiapan produksi.

### Functional Requirements
- **FR-7.1 Wishlist**: `GET/POST/DELETE /user/wishlists` (toggle per product), flag is_wishlisted di detail produk saat login.
- **FR-7.2 Banner**: CRUD `/admin/banners` (media, link, jadwal starts/ends, sort) + `GET /banners` publik (aktif & dalam jadwal, cache 5 menit).
- **FR-7.3 Report supplier**: `GET /supplier/reports/summary?from=&to=` {gmv, orders_count, aov, top_products[5], sales_per_day[]} dan `GET /supplier/reports/export` → enqueue `report:generate` → CSV ke storage → notifikasi tautan unduh (presigned, 24 jam).
- **FR-7.4 Report admin**: `GET /admin/reports/summary` {gmv, komisi, orders per status, supplier aktif, user baru, per-day series} + export CSV serupa.
- **FR-7.5 Hardening keamanan**: helmet-equivalent headers, batas body 2MB (kecuali upload), validasi UUID param terpusat, audit dependensi (`govulncheck`), rate limit khusus auth (5/menit), lockdown endpoint internal callback (secret + allowlist IP opsional).
- **FR-7.6 Observabilitas**: endpoint `/metrics` Prometheus (http duration/status, asynq queue depth via exporter), dashboard Grafana JSON di `deploy/`, log sampling untuk 4xx.
- **FR-7.7 Kualitas**: coverage usecase ≥70%; suite integration test E2E fase 5–6 dijalankan di CI (GitHub Actions: lint, test, build image); seed demo lengkap (`make seed-demo`: 3 supplier approved, 30 produk, 1 order tiap status) untuk demo investor.
- **FR-7.8 Dokumentasi**: `docs/openapi.yaml` sinkron seluruh endpoint publik/user/supplier/admin; README menjalankan project < 10 menit dari clone; PROGRESS.md final.

### Acceptance Criteria
- AC-7.a CI hijau end-to-end pada PR. `make seed-demo` + login 3 role → seluruh alur demo bisa diperagakan tanpa menyentuh DB manual.
- AC-7.b `/metrics` menampilkan histogram request; Grafana dashboard menampilkan RPS & p95 saat load test ringan (k6/hey 100 VU, 1 menit) tanpa error 5xx dan p95 < 300ms untuk katalog (mock eksternal).
- AC-7.c govulncheck bersih atau seluruh temuan tercatat dengan mitigasi; endpoint callback internal tanpa secret → 401.
- AC-7.d OpenAPI valid (lint) dan mencakup 100% route yang terdaftar (dibuktikan script pembanding route vs spec).

---

## Non-Functional Requirements (berlaku semua fase)

- **NFR-1** Semua timestamp `timestamptz` UTC; presentasi timezone urusan client.
- **NFR-2** Uang: `numeric(15,2)` di DB, integer rupiah di API (tanpa float di kalkulasi — gunakan int64 sen ATAU rupiah bulat; keputusan: rupiah bulat int64, tanpa sen).
- **NFR-3** Idempotensi wajib untuk: callback payment, callback disbursement, settlement, job yang bisa retry.
- **NFR-4** Query list apa pun wajib paginated (default 20, max 100) — tidak ada SELECT tanpa batas.
- **NFR-5** N+1 dilarang pada endpoint list (buktikan dengan log query saat review).
- **NFR-6** Secret hanya via env; log tidak boleh memuat password/OTP/token/no. rekening penuh (mask).
- **NFR-7** Semua external call (Xendit, RajaOngkir, Mailjet, storage) punya timeout eksplisit dan error terpetakan — kegagalan eksternal tidak boleh membuat data internal inkonsisten (gunakan kompensasi/outbox sederhana bila perlu).

## Definition of Done per Fase

1. Seluruh FR terimplementasi dan seluruh AC lolos (otomatis bila memungkinkan, manual didokumentasikan di PROGRESS.md).
2. `make lint && make test` hijau; migrasi up/down bersih dari database kosong.
3. Tidak ada TODO/panic placeholder di jalur produksi.
4. PROGRESS.md diperbarui: checklist FR & AC fase tercentang + catatan keputusan yang diambil bila ada ambiguitas.
