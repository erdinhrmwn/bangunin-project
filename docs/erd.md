# ERD & Process Flows — Building Materials Marketplace

All diagrams use Mermaid. The ERD is split per domain for readability; cross-domain relations are noted at the end of the ERD section.

---

## 1. ERD

### 1.1 Domain: User, Role & Supplier

```mermaid
erDiagram
    roles ||--o{ users : "has many"
    users ||--o{ user_addresses : "has many"
    users ||--o| suppliers : "has one (if role supplier)"
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
        varchar label "home | project | warehouse"
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
        int origin_city_id "shipping origin (RajaOngkir)"
        text pickup_address
        boolean own_fleet_enabled "own fleet"
        int fleet_coverage_km "fleet coverage range"
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

### 1.2 Domain: Catalog & Inventory

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
        jsonb specs "brand, material, dimensions, etc."
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
        varchar unit "sack | rod | m3 | box | pcs"
        numeric price
        int weight_gram "required - basis for shipping cost"
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
    order_items ||--o| reviews : "has one (after completed)"

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
        numeric commission_amount "platform commission"
        numeric total
        jsonb shipping_address "address snapshot"
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

### 1.4 Domain: Finance (Payout), Notifications & Others

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
        uuid supplier_id PK, FK
        numeric available_balance
        numeric pending_balance "order not yet completed"
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
        timestamptz created_at "append-only, no update/delete"
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

### 1.5 Cross-Domain Relations (summary)

- `suppliers.user_id → users.id` (1:1, unique — one account, one role per the `role_id` decision in `users`)
- `products.supplier_id → suppliers.id`
- `cart_items.variant_id → product_variants.id`
- `orders.supplier_id → suppliers.id`, `orders.user_id → users.id`
- `stock_reservations.order_id → orders.id`
- `ledger_entries.ref_id → orders.id / withdraw_requests.id`
- `reviews.product_id → products.id` (rating aggregate stored denormalized in `products`)

---

## 2. Process Flows

### 2.1 Supplier Registration & Approval

```mermaid
flowchart TD
    A[User registers account\nrole: supplier] --> B[Email OTP verification]
    B --> C[Complete store profile:\nname, address, origin city,\nbank account]
    C --> D[Upload documents:\nNIB / KTP / NPWP]
    D --> E[Submit application\nstatus: pending]
    E --> F{Admin review}
    F -- Approve --> G[status: approved\nverified_at set]
    G --> H[Email + in-app notification\nSupplier can start selling]
    F -- Reject --> I[status: rejected\n+ rejection_note]
    I --> J[Notification of rejection reason]
    J --> C
    G -.-> K[audit_logs:\nsupplier.approve]
    I -.-> K2[audit_logs:\nsupplier.reject]
```

### 2.2 Checkout & Payment (end-to-end)

```mermaid
sequenceDiagram
    autonumber
    actor U as User
    participant API as Monolith API
    participant RD as Redis
    participant DB as PostgreSQL
    participant RO as RajaOngkir
    participant XD as Xendit

    U->>API: POST /checkout/preview (cart items, address)
    API->>DB: Validate stock, price, min_order_qty
    API->>API: Group items per supplier (split order)
    loop per supplier
        API->>RO: Calculate shipping cost (supplier origin → destination,\ntotal weight vs volumetric)
        RO-->>API: List of couriers + rates
    end
    API-->>U: Summary: N orders, subtotal, courier options per order

    U->>API: POST /checkout/confirm (courier selection)
    API->>RD: Acquire lock on stock per variant
    API->>DB: BEGIN TX
    API->>DB: Re-validate price & stock
    API->>DB: INSERT checkout_group + orders (per supplier)\n+ order_items (snapshot) + stock_reservations
    API->>DB: COMMIT
    API->>RD: Release lock
    API->>XD: Create Invoice API (via PaymentService)
    XD-->>API: invoice_id + invoice_url
    API->>DB: INSERT payments (pending)
    API-->>U: Redirect to Xendit invoice_url

    U->>XD: Pay (VA / e-wallet / QRIS)
    XD->>API: Webhook: invoice.paid (POST /internal/payments/callback)
    API->>DB: payments.status = paid,\norders.status = paid,\nconvert stock_reservations → stock_movements
    API->>API: Enqueue job email:send + in-app notification
    API-->>U: "Payment successful" notification
```

If not paid by `expires_at`, the `order:expire` job in the worker will change the status to `expired`, release all `stock_reservations`, and send a notification.

### 2.3 Order Status State Machine

```mermaid
stateDiagram-v2
    [*] --> pending_payment : checkout confirm
    pending_payment --> paid : Xendit webhook (system)
    pending_payment --> expired : order-expire job (system)
    pending_payment --> cancelled : cancelled by user
    paid --> processed : supplier processes order
    paid --> cancelled : cancelled (refund via payment service)
    processed --> shipped : supplier enters tracking number /\nown fleet schedule
    shipped --> delivered : tracking delivered /\nfleet confirmation
    delivered --> completed : user confirmation /\nauto-complete job after N days (system)
    completed --> [*]

    note right of completed
        Trigger settlement:
        credit supplier balance
        (subtotal + own fleet shipping cost
        - platform commission)
        via ledger_entries
    end note

    note right of cancelled
        If already paid:
        refund via payment service
        + release/return stock
    end note
```

Actor rules per transition: `paid → processed → shipped` supplier only; `delivered → completed` by user or system; `cancelled` follows the rules above; admin can intervene with logging in `audit_logs`. All transitions are recorded in `order_status_histories`.

### 2.4 Stock Reservation (lifecycle)

```mermaid
flowchart LR
    A[Checkout confirm] --> B[Redis lock per variant]
    B --> C{Stock sufficient?}
    C -- No --> D[Reject: ErrInsufficientStock]
    C -- Yes --> E[INSERT stock_reservations\nstatus: active, expires_at = +2 hours]
    E --> F{Payment result}
    F -- Paid --> G[status: converted\nINSERT stock_movements type out\nUPDATE variant.stock]
    F -- Expired / cancelled --> H[status: released\nstock available again]
```

### 2.5 Supplier Payout / Withdraw

```mermaid
sequenceDiagram
    autonumber
    actor S as Supplier
    actor AD as Admin
    participant API as Monolith API
    participant DB as PostgreSQL
    participant XD as Xendit

    Note over API,DB: Order completed → system credits balance
    API->>DB: ledger_entries: credit_order + debit_commission,\nupdate supplier_balances.available

    S->>API: POST /supplier/withdraws (amount, bank_account)
    API->>DB: Validate sufficient balance → INSERT withdraw_requests (requested),\nhold balance (available → pending)
    API-->>S: Request submitted

    AD->>API: Approve withdraw
    API->>DB: status = approved → processing
    API->>XD: Disbursement API (via PaymentService)
    XD-->>API: Webhook: disbursement completed
    API->>DB: status = disbursed,\nledger_entries: debit_withdraw,\nrelease hold
    API-->>S: Notification: funds sent

    alt Rejected by admin / disbursement failed
        API->>DB: status = rejected/failed,\nreturn hold to available
        API-->>S: Notification + reason
    end
```

### 2.6 Product Publishing by Supplier

```mermaid
flowchart TD
    A[Supplier creates product\nstatus: draft] --> B[Fill in variant: price, unit,\nweight, dimensions, min order, stock]
    B --> C[Upload images\n→ worker resize/compress]
    C --> D{Validation complete?\nmin 1 variant + 1 image\n+ weight filled}
    D -- No --> B
    D -- Yes --> E[Publish → status: active]
    E --> F[search_vector updated\nenters public catalog]
    E -.-> G{Admin moderation\nif reported}
    G -- Violation --> H[status: banned\n+ supplier notification]
```

---

## 3. Implementation Notes Related to the Diagrams

- **Snapshots in `order_items` and `orders.shipping_address`** ensure that price/address changes after checkout do not alter historical order data.
- **One `payments` per `checkout_group`** — Xendit receives a single invoice for the entire group; fund distribution to suppliers happens at the ledger level, not at the payment level.
- **`ledger_entries` append-only** with `balance_after` makes balances reconstructable and auditable at any time; `supplier_balances` is merely a materialized snapshot.
- **`search_vector` (tsvector)** populated via trigger or on product update, GIN-indexed for full-text search without needing Elasticsearch in the early phase.
- All these diagrams are consistent with prior decisions: `role_id` FK directly on `users`, single monolith with internal adapters for payment/notification/shipping (no separate services), no voucher & unit converter.
