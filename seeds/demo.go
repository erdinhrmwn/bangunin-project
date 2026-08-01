package seeds

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/pkg/hash"
)

const demoPassword = "Password123!"

type demoVariant struct {
	id         uuid.UUID
	supplierID uuid.UUID
	price      float64
	weightGram int
}

// Demo seeds a full demo dataset for manual QA / phase verification (FR-7.7):
// 3 approved suppliers with 30 products (1 variant each), and 1 order per
// order status. Idempotent via upsert-on-natural-key + RETURNING id, safe to
// rerun (cmd/migrate -seed -demo).
func Demo(ctx context.Context, db *pgxpool.Pool) error {
	categoryID, err := demoCategory(ctx, db)
	if err != nil {
		return err
	}

	buyerID, err := demoUser(ctx, db, "demo-buyer@bangunin.test", "Demo Buyer", entity.RoleUserID)
	if err != nil {
		return err
	}

	supplierNames := []string{"Toko Bangunan Jaya", "Sumber Material Makmur", "Distributor Semen Sentosa"}
	var variants []demoVariant
	for si, name := range supplierNames {
		supplierID, err := demoSupplier(ctx, db, si, name)
		if err != nil {
			return err
		}
		for pi := 0; pi < 10; pi++ {
			v, err := demoProduct(ctx, db, supplierID, categoryID, si, pi)
			if err != nil {
				return err
			}
			variants = append(variants, v)
		}
	}

	statuses := []string{
		entity.OrderStatusPendingPayment, entity.OrderStatusPaid, entity.OrderStatusProcessed,
		entity.OrderStatusShipped, entity.OrderStatusDelivered, entity.OrderStatusCompleted,
		entity.OrderStatusExpired, entity.OrderStatusCancelled,
	}
	for i, status := range statuses {
		if err := demoOrder(ctx, db, buyerID, variants[i%len(variants)], i, status); err != nil {
			return err
		}
	}
	return nil
}

func demoCategory(ctx context.Context, db *pgxpool.Pool) (int, error) {
	var id int
	err := db.QueryRow(ctx,
		`INSERT INTO categories (name, slug) VALUES ($1, $2)
		 ON CONFLICT (slug) DO UPDATE SET slug = EXCLUDED.slug
		 RETURNING id`,
		"Material Bangunan", "material-bangunan",
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("seeds: demo category: %w", err)
	}
	return id, nil
}

func demoUser(ctx context.Context, db *pgxpool.Pool, email, name string, roleID int16) (uuid.UUID, error) {
	hashed, err := hash.Hash(demoPassword)
	if err != nil {
		return uuid.UUID{}, err
	}
	id := uuid.Must(uuid.NewV7())
	err = db.QueryRow(ctx,
		`INSERT INTO users (id, role_id, name, email, password_hash, status, email_verified_at)
		 VALUES ($1, $2, $3, $4, $5, 'active', now())
		 ON CONFLICT (email) DO UPDATE SET email = EXCLUDED.email
		 RETURNING id`,
		id, roleID, name, email, hashed,
	).Scan(&id)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("seeds: demo user %s: %w", email, err)
	}
	return id, nil
}

func demoSupplier(ctx context.Context, db *pgxpool.Pool, idx int, storeName string) (uuid.UUID, error) {
	userID, err := demoUser(ctx, db, fmt.Sprintf("demo-supplier%d@bangunin.test", idx+1), storeName, entity.RoleSupplierID)
	if err != nil {
		return uuid.UUID{}, err
	}
	slug := fmt.Sprintf("demo-supplier-%d", idx+1)
	id := uuid.Must(uuid.NewV7())
	err = db.QueryRow(ctx,
		`INSERT INTO suppliers (id, user_id, store_name, slug, status, origin_city_id, verified_at)
		 VALUES ($1, $2, $3, $4, 'approved', 154, now())
		 ON CONFLICT (slug) DO UPDATE SET slug = EXCLUDED.slug
		 RETURNING id`,
		id, userID, storeName, slug,
	).Scan(&id)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("seeds: demo supplier %s: %w", storeName, err)
	}
	return id, nil
}

func demoProduct(ctx context.Context, db *pgxpool.Pool, supplierID uuid.UUID, categoryID, si, pi int) (demoVariant, error) {
	n := si*10 + pi + 1
	name := fmt.Sprintf("Produk Demo %d", n)
	slug := fmt.Sprintf("produk-demo-%d", n)
	productID := uuid.Must(uuid.NewV7())
	err := db.QueryRow(ctx,
		`INSERT INTO products (id, supplier_id, category_id, name, slug, status)
		 VALUES ($1, $2, $3, $4, $5, 'active')
		 ON CONFLICT (slug) DO UPDATE SET slug = EXCLUDED.slug
		 RETURNING id`,
		productID, supplierID, categoryID, name, slug,
	).Scan(&productID)
	if err != nil {
		return demoVariant{}, fmt.Errorf("seeds: demo product %s: %w", slug, err)
	}

	price := float64(50_000 + n*1_000)
	weightGram := 40_000
	variantID := uuid.Must(uuid.NewV7())
	sku := fmt.Sprintf("DEMO-%d", n)
	err = db.QueryRow(ctx,
		`INSERT INTO product_variants (id, product_id, supplier_id, sku, name, unit, price, weight_gram, min_order_qty, stock, is_active)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 1, 100, true)
		 ON CONFLICT (supplier_id, sku) DO UPDATE SET sku = EXCLUDED.sku
		 RETURNING id`,
		variantID, productID, supplierID, sku, name+" - Default", entity.UnitSak, price, weightGram,
	).Scan(&variantID)
	if err != nil {
		return demoVariant{}, fmt.Errorf("seeds: demo variant %s: %w", sku, err)
	}
	return demoVariant{id: variantID, supplierID: supplierID, price: price, weightGram: weightGram}, nil
}

func demoOrder(ctx context.Context, db *pgxpool.Pool, buyerID uuid.UUID, v demoVariant, idx int, status string) error {
	cgStatus := entity.CheckoutGroupStatusPendingPayment
	if status != entity.OrderStatusPendingPayment && status != entity.OrderStatusExpired && status != entity.OrderStatusCancelled {
		cgStatus = entity.CheckoutGroupStatusPaid
	}
	const qty = 2
	subtotal := v.price * qty
	shippingCost := 25_000.0
	commission := subtotal * 0.04
	total := subtotal + shippingCost

	groupNumber := fmt.Sprintf("DEMO-CG-%d", idx+1)
	groupID := uuid.Must(uuid.NewV7())
	err := db.QueryRow(ctx,
		`INSERT INTO checkout_groups (id, user_id, group_number, grand_total, status)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (group_number) DO UPDATE SET group_number = EXCLUDED.group_number
		 RETURNING id`,
		groupID, buyerID, groupNumber, total, cgStatus,
	).Scan(&groupID)
	if err != nil {
		return fmt.Errorf("seeds: demo checkout_group %s: %w", groupNumber, err)
	}

	addr := entity.ShippingAddressSnapshot{
		RecipientName: "Demo Buyer", RecipientPhone: "081234567890",
		ProvinceID: 6, CityID: 154, Subdistrict: "Cilandak", PostalCode: "12430",
		AddressDetail: "Jl. Demo No. 1",
	}
	addrJSON, err := json.Marshal(addr)
	if err != nil {
		return fmt.Errorf("seeds: demo order address: %w", err)
	}

	orderNumber := fmt.Sprintf("DEMO-ORD-%d", idx+1)
	orderID := uuid.Must(uuid.NewV7())
	err = db.QueryRow(ctx,
		`INSERT INTO orders (id, checkout_group_id, user_id, supplier_id, order_number, status, subtotal, shipping_cost, commission_amount, total, shipping_address)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 ON CONFLICT (order_number) DO UPDATE SET order_number = EXCLUDED.order_number
		 RETURNING id`,
		orderID, groupID, buyerID, v.supplierID, orderNumber, status, subtotal, shippingCost, commission, total, addrJSON,
	).Scan(&orderID)
	if err != nil {
		return fmt.Errorf("seeds: demo order %s: %w", orderNumber, err)
	}

	var hasItem bool
	if err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM order_items WHERE order_id = $1)`, orderID).Scan(&hasItem); err != nil {
		return fmt.Errorf("seeds: demo order_item check for %s: %w", orderNumber, err)
	}
	if hasItem {
		return nil
	}
	_, err = db.Exec(ctx,
		`INSERT INTO order_items (id, order_id, variant_id, product_name, variant_name, unit, price, qty, weight_gram)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		uuid.Must(uuid.NewV7()), orderID, v.id, "Demo Product", "Demo Variant - Default", entity.UnitSak, v.price, qty, v.weightGram,
	)
	if err != nil {
		return fmt.Errorf("seeds: demo order_item for %s: %w", orderNumber, err)
	}
	return nil
}
