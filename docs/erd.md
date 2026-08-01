# ERD & Alur Proses — Marketplace Bahan Bangunan

Semua diagram menggunakan Mermaid. ERD dipecah per domain agar mudah dibaca; relasi lintas domain dicatat di bagian akhir ERD.

---

## 1. ERD

### 1.1 Domain: User, Role & Supplier

```mermaid
erDiagram
    roles ||--o{ users : "has many"
    users ||--o{ user_addresses : "has many"
    users ||--o| suppliers : "has one (jika role supplier)"
    suppliers ||--o{ supplier_documents : "has many"
    suppliers ||--o{ supplier_bank_accounts : "has many"

    roles {
        smallint id PK
        varchar name "admin | supplier | user"
    }

    users {
        uuid id PK
        smallint role_id FK
        varchar name
        varchar email UK
        varchar password_hash
        varchar phone
        varchar status "active | suspended | banned"
        timestamptz email_verified_at
        timestamptz created_at
    }

    user_addresses {
        uuid id PK
        uuid user_id FK
        varchar label "rumah | proyek | gudang"
        varchar recipient_name
        varchar recipient_phone
        int province_id "ref RajaOngkir"
        int city_id "ref RajaOngkir"
        varchar subdistrict
        varchar postal_code
        text address_detail
        boolean is_default
    }

    suppliers {
        uuid id PK
        uuid user_id FK "UNIQUE"
        varchar store_name
        varchar slug UK
        text description
        varchar status "pending | approved | rejected | suspended"
        int origin_city_id "origin ongkir (RajaOngkir)"
        text pickup_address
        boolean own_fleet_enabled "armada sendiri"
        int fleet_coverage_km "jangkauan armada"
        numeric fleet_flat_rate
        timestamptz verified_at
    }

    supplier_documents {
        uuid id PK
        uuid supplier_id FK
        varchar doc_type "nib | ktp | npwp"
        varchar file_url
        varchar status "pending | approved | rejected"
        text rejection_note
    }

    supplier_bank_accounts {
        uuid id PK
        uuid supplier_id FK
        varchar bank_code
        varchar account_number
        varchar account_name
        boolean is_default
    }
```

### 1.2 Domain: Katalog & Inventory

```mermaid
erDiagram
    categories ||--o{ categories : "parent-child"
    categories ||--o{ products : "has many"
    suppliers ||--o{ products : "has many"
    products ||--o{ product_variants : "has many"
    products ||--o{ product_images : "has many"
    product_variants ||--o{ stock_movements : "has many"
    product_variants ||--o{ stock_reservations : "has many"

    categories {
        int id PK
        int parent_id FK "nullable"
        varchar name
        varchar slug UK
        int sort_order
        boolean is_active
    }

    products {
        uuid id PK
        uuid supplier_id FK
        int category_id FK
        varchar name
        varchar slug UK
        text description
        jsonb specs "merk, material, dimensi, dll"
        varchar status "draft | active | inactive | banned"
        numeric rating_avg
        int rating_count
        tsvector search_vector "GIN index"
        timestamptz created_at
    }

    product_variants {
        uuid id PK
        uuid product_id FK
        varchar name "40kg | 50kg | 6m | 12m"
        varchar sku UK
        varchar unit "sak | batang | m3 | dus | pcs"
        numeric price
        int weight_gram "wajib - dasar ongkir"
        int length_cm
        int width_cm
        int height_cm
        int min_order_qty
        int stock
        boolean is_active
    }

    product_images {
        uuid id PK
        uuid product_id FK
        varchar url
        boolean is_primary
        int sort_order
    }

    stock_movements {
        uuid id PK
        uuid variant_id FK
        varchar type "in | out | adjustment | reserve_convert"
        int qty "+/-"
        varchar ref_type "order | manual | import"
        uuid ref_id
        int stock_after
        text note
        timestamptz created_at
    }

    stock_reservations {
        uuid id PK
        uuid variant_id FK
        uuid order_id FK
        int qty
        varchar status "active | converted | released"
        timestamptz expires_at
    }
```

### 1.3 Domain: Cart, Checkout, Order & Shipping

```mermaid
erDiagram
    users ||--o| carts : "has one"
    carts ||--o{ cart_items : "has many"
    users ||--o{ checkout_groups : "has many"
    checkout_groups ||--o{ orders : "split per supplier"
    checkout_groups ||--o| payments : "has one"
    orders ||--o{ order_items : "has many"
    orders ||--o{ order_status_histories : "has many"
    orders ||--o| shipments : "has one"
    order_items ||--o| reviews : "has one (setelah completed)"

    carts {
        uuid id PK
        uuid user_id FK "UNIQUE"
        timestamptz updated_at
    }

    cart_items {
        uuid id PK
        uuid cart_id FK
        uuid variant_id FK
        int qty
    }

    checkout_groups {
        uuid id PK
        uuid user_id FK
        varchar group_number UK
        numeric grand_total
        varchar status "pending_payment | paid | expired | cancelled"
        timestamptz created_at
    }

    payments {
        uuid id PK
        uuid checkout_group_id FK "UNIQUE"
        varchar xendit_invoice_id UK
        varchar method "va | ewallet | qris"
        numeric amount
        varchar status "pending | paid | expired | failed"
        varchar invoice_url
        timestamptz paid_at
        timestamptz expires_at
    }

    orders {
        uuid id PK
        uuid checkout_group_id FK
        uuid user_id FK
        uuid supplier_id FK
        varchar order_number UK
        varchar status "pending_payment .. completed"
        numeric subtotal
        numeric shipping_cost
        numeric commission_amount "komisi platform"
        numeric total
        jsonb shipping_address "snapshot alamat"
        timestamptz created_at
    }

    order_items {
        uuid id PK
        uuid order_id FK
        uuid variant_id FK
        varchar product_name "snapshot"
        varchar variant_name "snapshot"
        varchar unit "snapshot"
        numeric price "snapshot"
        int qty
        int weight_gram "snapshot"
    }

    order_status_histories {
        uuid id PK
        uuid order_id FK
        varchar from_status
        varchar to_status
        varchar actor_type "user | supplier | admin | system"
        uuid actor_id
        text note
        timestamptz created_at
    }

    shipments {
        uuid id PK
        uuid order_id FK "UNIQUE"
        varchar method "courier | supplier_fleet"
        varchar courier_code "jne | sicepat | null"
        varchar courier_service
        varchar tracking_number
        numeric cost
        varchar status "pending | picked_up | in_transit | delivered"
        timestamptz shipped_at
        timestamptz delivered_at
    }

    reviews {
        uuid id PK
        uuid order_item_id FK "UNIQUE"
        uuid product_id FK
        uuid user_id FK
        smallint rating "1-5"
        text comment
        text supplier_reply
        timestamptz reply_at
        timestamptz created_at
    }
```

### 1.4 Domain: Finance (Payout), Notifikasi & Lainnya

```mermaid
erDiagram
    suppliers ||--|| supplier_balances : "has one"
    suppliers ||--o{ ledger_entries : "has many"
    suppliers ||--o{ withdraw_requests : "has many"
    supplier_bank_accounts ||--o{ withdraw_requests : "target"
    users ||--o{ notifications : "has many"
    users ||--o{ wishlists : "has many"
    users ||--o{ audit_logs : "actor"

    supplier_balances {
        uuid supplier_id PK_FK
        numeric available_balance
        numeric pending_balance "order belum completed"
        timestamptz updated_at
    }

    ledger_entries {
        uuid id PK
        uuid supplier_id FK
        varchar entry_type "credit_order | debit_commission | debit_withdraw | credit_refund_reversal"
        numeric amount
        varchar ref_type "order | withdraw"
        uuid ref_id
        numeric balance_after
        timestamptz created_at "append-only, tanpa update/delete"
    }

    withdraw_requests {
        uuid id PK
        uuid supplier_id FK
        uuid bank_account_id FK
        numeric amount
        varchar status "requested | approved | processing | disbursed | rejected | failed"
        uuid processed_by "admin user_id"
        varchar disbursement_id "ref payment service"
        text rejection_note
        timestamptz created_at
    }

    notifications {
        uuid id PK
        uuid user_id FK
        varchar type "order_update | approval | payout | system"
        varchar title
        text body
        jsonb data "deeplink payload"
        timestamptz read_at
        timestamptz created_at
    }

    wishlists {
        uuid id PK
        uuid user_id FK
        uuid product_id FK
    }

    audit_logs {
        uuid id PK
        uuid actor_id FK
        varchar action "supplier.approve | product.ban | withdraw.approve"
        varchar entity_type
        uuid entity_id
        jsonb metadata "before/after"
        varchar ip_address
        timestamptz created_at
    }

    banners {
        uuid id PK
        varchar title
        varchar image_url
        varchar link_url
        boolean is_active
        int sort_order
        timestamptz starts_at
        timestamptz ends_at
    }
```

### 1.5 Relasi Lintas Domain (ringkasan)

- `suppliers.user_id → users.id` (1:1, unik — satu akun satu peran sesuai keputusan `role_id` di `users`)
- `products.supplier_id → suppliers.id`
- `cart_items.variant_id → product_variants.id`
- `orders.supplier_id → suppliers.id`, `orders.user_id → users.id`
- `stock_reservations.order_id → orders.id`
- `ledger_entries.ref_id → orders.id / withdraw_requests.id`
- `reviews.product_id → products.id` (agregat rating disimpan denormalized di `products`)

---

## 2. Alur Proses

### 2.1 Registrasi & Approval Supplier

```mermaid
flowchart TD
    A[User register akun\nrole: supplier] --> B[Verifikasi email OTP]
    B --> C[Lengkapi profil toko:\nnama, alamat, origin city,\nrekening bank]
    C --> D[Upload dokumen:\nNIB / KTP / NPWP]
    D --> E[Submit pengajuan\nstatus: pending]
    E --> F{Review oleh Admin}
    F -- Approve --> G[status: approved\nverified_at diisi]
    G --> H[Notifikasi email + in-app\nSupplier bisa mulai jualan]
    F -- Reject --> I[status: rejected\n+ rejection_note]
    I --> J[Notifikasi alasan penolakan]
    J --> C
    G -.-> K[audit_logs:\nsupplier.approve]
    I -.-> K2[audit_logs:\nsupplier.reject]
```

### 2.2 Checkout & Pembayaran (end-to-end)

```mermaid
sequenceDiagram
    autonumber
    actor U as User
    participant API as Monolith API
    participant RD as Redis
    participant DB as PostgreSQL
    participant RO as RajaOngkir
    participant PS as Payment Service (gRPC)
    participant XD as Xendit

    U->>API: POST /checkout/preview (cart items, address)
    API->>DB: Validasi stok, harga, min_order_qty
    API->>API: Group item per supplier (split order)
    loop per supplier
        API->>RO: Hitung ongkir (origin supplier → tujuan,\nberat total vs volumetrik)
        RO-->>API: Daftar kurir + tarif
    end
    API-->>U: Ringkasan: N order, subtotal, opsi kurir per order

    U->>API: POST /checkout/confirm (pilihan kurir)
    API->>RD: Acquire lock stok per variant
    API->>DB: BEGIN TX
    API->>DB: Validasi ulang harga & stok
    API->>DB: INSERT checkout_group + orders (per supplier)\n+ order_items (snapshot) + stock_reservations
    API->>DB: COMMIT
    API->>RD: Release lock
    API->>PS: CreateInvoice(checkout_group, amount) [gRPC]
    PS->>XD: Create Invoice API
    XD-->>PS: invoice_id + invoice_url
    PS-->>API: invoice_url, expires_at
    API->>DB: INSERT payments (pending)
    API-->>U: Redirect ke invoice_url Xendit

    U->>XD: Bayar (VA / e-wallet / QRIS)
    XD->>PS: Webhook: invoice.paid
    PS->>API: PaymentPaid(checkout_group_id) [gRPC]
    API->>DB: payments.status = paid,\norders.status = paid,\nkonversi stock_reservations → stock_movements
    API->>API: Enqueue job email:send + notifikasi in-app
    API-->>U: Notifikasi "Pembayaran berhasil"
```

Jika tidak dibayar sampai `expires_at`, job `order:expire` di worker akan mengubah status menjadi `expired`, me-release semua `stock_reservations`, dan mengirim notifikasi.

### 2.3 State Machine Status Order

```mermaid
stateDiagram-v2
    [*] --> pending_payment : checkout confirm
    pending_payment --> paid : webhook Xendit (system)
    pending_payment --> expired : job order-expire (system)
    pending_payment --> cancelled : dibatalkan user
    paid --> processed : supplier proses pesanan
    paid --> cancelled : dibatalkan (refund via payment service)
    processed --> shipped : supplier input resi /\njadwal armada sendiri
    shipped --> delivered : tracking delivered /\nkonfirmasi armada
    delivered --> completed : konfirmasi user /\njob auto-complete N hari (system)
    completed --> [*]

    note right of completed
        Trigger settlement:
        credit saldo supplier
        (subtotal + ongkir armada sendiri
        - komisi platform)
        via ledger_entries
    end note

    note right of cancelled
        Jika sudah paid:
        refund via payment service
        + release/kembalikan stok
    end note
```

Aturan aktor per transisi: `paid → processed → shipped` hanya supplier; `delivered → completed` oleh user atau system; `cancelled` mengikuti aturan di atas; admin dapat melakukan intervensi dengan pencatatan di `audit_logs`. Semua transisi dicatat di `order_status_histories`.

### 2.4 Reservasi Stok (lifecycle)

```mermaid
flowchart LR
    A[Checkout confirm] --> B[Redis lock per variant]
    B --> C{Stok cukup?}
    C -- Tidak --> D[Tolak: ErrInsufficientStock]
    C -- Ya --> E[INSERT stock_reservations\nstatus: active, expires_at = +2 jam]
    E --> F{Hasil pembayaran}
    F -- Dibayar --> G[status: converted\nINSERT stock_movements type out\nUPDATE variant.stock]
    F -- Expired / dibatalkan --> H[status: released\nstok kembali tersedia]
```

### 2.5 Payout / Withdraw Supplier

```mermaid
sequenceDiagram
    autonumber
    actor S as Supplier
    actor AD as Admin
    participant API as Monolith API
    participant DB as PostgreSQL
    participant PS as Payment Service (gRPC)
    participant XD as Xendit

    Note over API,DB: Order completed → system kredit saldo
    API->>DB: ledger_entries: credit_order + debit_commission,\nupdate supplier_balances.available

    S->>API: POST /supplier/withdraws (amount, bank_account)
    API->>DB: Validasi saldo cukup → INSERT withdraw_requests (requested),\nhold saldo (available → pending)
    API-->>S: Request terkirim

    AD->>API: Approve withdraw
    API->>DB: status = approved → processing
    API->>PS: CreateDisbursement(bank, amount) [gRPC]
    PS->>XD: Disbursement API
    XD-->>PS: Callback: disbursement completed
    PS->>API: DisbursementCompleted [gRPC]
    API->>DB: status = disbursed,\nledger_entries: debit_withdraw,\nrelease hold
    API-->>S: Notifikasi dana dikirim

    alt Ditolak admin / gagal disbursement
        API->>DB: status = rejected/failed,\nkembalikan hold ke available
        API-->>S: Notifikasi + alasan
    end
```

### 2.6 Publikasi Produk oleh Supplier

```mermaid
flowchart TD
    A[Supplier buat produk\nstatus: draft] --> B[Isi varian: harga, satuan,\nberat, dimensi, min order, stok]
    B --> C[Upload gambar\n→ worker resize/compress]
    C --> D{Validasi lengkap?\nmin 1 varian + 1 gambar\n+ berat terisi}
    D -- Tidak --> B
    D -- Ya --> E[Publish → status: active]
    E --> F[search_vector terupdate\nmasuk katalog publik]
    E -.-> G{Moderasi admin\njika dilaporkan}
    G -- Melanggar --> H[status: banned\n+ notifikasi supplier]
```

---

## 3. Catatan Implementasi Terkait Diagram

- **Snapshot di `order_items` dan `orders.shipping_address`** memastikan perubahan harga/alamat setelah checkout tidak mengubah data historis order.
- **Satu `payments` per `checkout_group`** — Xendit menerima satu invoice untuk seluruh grup; pembagian dana ke supplier terjadi di level ledger, bukan di level pembayaran.
- **`ledger_entries` append-only** dengan `balance_after` membuat saldo bisa direkonstruksi dan diaudit kapan pun; `supplier_balances` hanyalah materialized snapshot.
- **`search_vector` (tsvector)** diisi via trigger atau saat update produk, di-index GIN untuk full-text search tanpa perlu Elasticsearch di fase awal.
- Semua diagram ini konsisten dengan keputusan sebelumnya: `role_id` FK langsung di `users`, monolith + 2 service gRPC, tanpa voucher & unit converter.
