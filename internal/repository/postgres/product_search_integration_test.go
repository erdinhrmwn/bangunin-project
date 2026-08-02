//go:build integration

// Package postgres_test holds testcontainers-backed integration tests
// (AC-4.e). Run with: go test -tags=integration ./internal/repository/postgres/...
package postgres_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	migratepg "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/lib/pq"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"erdinhrmwn/bangunin/config"
	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/repository"
	"erdinhrmwn/bangunin/internal/infra/database"
	"erdinhrmwn/bangunin/migrations"

	postgresrepo "erdinhrmwn/bangunin/internal/repository/postgres"
)

// setupDB spins up an ephemeral postgres container and runs all embedded
// migrations against it, returning a connected pool.
func setupDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	ctr, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("bangunin_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(wait.ForListeningPort("5432/tcp").WithStartupTimeout(30*time.Second)),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = testcontainers.TerminateContainer(ctr) })

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	sdb, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer sdb.Close()

	driver, err := migratepg.WithInstance(sdb, &migratepg.Config{})
	require.NoError(t, err)
	src, err := iofs.New(migrations.FS, ".")
	require.NoError(t, err)
	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	require.NoError(t, err)
	require.NoError(t, m.Up())

	pool, err := database.NewPool(ctx, config.DBConfig{DSN: dsn, MaxConns: 5, MinConns: 1, MaxConnLife: time.Hour})
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return pool
}

// seedFixture holds one supplier (approved or suspended) + a category, used
// as the base for seeding N products.
type seedFixture struct {
	pool        *pgxpool.Pool
	categoryID  int
	supplierID  uuid.UUID // approved
	suspendedID uuid.UUID // suspended supplier — AC-4.d exclusion check
}

func seedBase(t *testing.T, pool *pgxpool.Pool) seedFixture {
	t.Helper()
	ctx := context.Background()

	_, err := pool.Exec(ctx, `INSERT INTO roles (id, name) VALUES (2, 'supplier')`)
	require.NoError(t, err)

	var categoryID int
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO categories (name, slug) VALUES ('Semen', 'semen') RETURNING id`,
	).Scan(&categoryID))

	mkSupplier := func(status, storeName, slug string) uuid.UUID {
		userID := uuid.Must(uuid.NewV7())
		_, err := pool.Exec(ctx,
			`INSERT INTO users (id, role_id, name, email, password_hash) VALUES ($1, 2, $2, $3, 'x')`,
			userID, storeName, storeName+"@example.com",
		)
		require.NoError(t, err)

		supplierID := uuid.Must(uuid.NewV7())
		_, err = pool.Exec(ctx,
			`INSERT INTO suppliers (id, user_id, store_name, slug, status, origin_city_id)
			 VALUES ($1, $2, $3, $4, $5, 501)`,
			supplierID, userID, storeName, slug, status,
		)
		require.NoError(t, err)
		return supplierID
	}

	return seedFixture{
		pool:        pool,
		categoryID:  categoryID,
		supplierID:  mkSupplier(entity.SupplierStatusApproved, "Toko Approved", "toko-approved"),
		suspendedID: mkSupplier(entity.SupplierStatusSuspended, "Toko Suspended", "toko-suspended"),
	}
}

// seedProduct inserts one active product with one active variant (price,
// stock) and one primary image under the given supplier.
func (f seedFixture) seedProduct(t *testing.T, supplierID uuid.UUID, name, slug string, price float64, status string) uuid.UUID {
	t.Helper()
	ctx := context.Background()

	productID := uuid.Must(uuid.NewV7())
	_, err := f.pool.Exec(ctx,
		`INSERT INTO products (id, supplier_id, category_id, name, slug, description, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		productID, supplierID, f.categoryID, name, slug, "Semen berkualitas tinggi", status,
	)
	require.NoError(t, err)

	variantID := uuid.Must(uuid.NewV7())
	_, err = f.pool.Exec(ctx,
		`INSERT INTO product_variants (id, product_id, supplier_id, sku, name, unit, price, weight_gram, stock, is_active)
		 VALUES ($1, $2, $3, $4, $5, 'sak', $6, 40000, 10, true)`,
		variantID, productID, supplierID, slug+"-sku", name, price,
	)
	require.NoError(t, err)

	_, err = f.pool.Exec(ctx,
		`INSERT INTO product_images (id, product_id, url, is_primary) VALUES ($1, $2, $3, true)`,
		uuid.Must(uuid.NewV7()), productID, "https://example.com/"+slug+".jpg",
	)
	require.NoError(t, err)

	return productID
}

// AC-4.a + AC-4.e: seed 50 active products (varying price/name) plus a few
// draft/inactive/suspended-supplier ones that must NOT appear, then assert
// search text match, price filter, sort, and cursor pagination.
func TestProductRepository_Search(t *testing.T) {
	pool := setupDB(t)
	f := seedBase(t, pool)
	repo := postgresrepo.NewProductRepository(pool)
	ctx := context.Background()

	for i := 0; i < 48; i++ {
		f.seedProduct(t, f.supplierID, fmt.Sprintf("Bata Ringan %02d", i), fmt.Sprintf("bata-ringan-%02d", i), float64(10000+i*1000), entity.ProductStatusActive)
	}
	f.seedProduct(t, f.supplierID, "Portland Cement 40 Kg", "portland-cement-40kg", 65000, entity.ProductStatusActive)
	f.seedProduct(t, f.supplierID, "Semen Portland Merah Putih", "semen-portland-merah-putih", 62000, entity.ProductStatusActive)
	// Must be excluded: not active, and not-approved supplier.
	f.seedProduct(t, f.supplierID, "Draft Product", "draft-product", 50000, entity.ProductStatusDraft)
	f.seedProduct(t, f.suspendedID, "Suspended Supplier Product", "suspended-supplier-product", 50000, entity.ProductStatusActive)

	t.Run("AC-4.a text search finds published product", func(t *testing.T) {
		res, _, err := repo.Search(ctx, repository.SearchParams{Query: "semen"})
		require.NoError(t, err)
		require.NotEmpty(t, res)
		found := false
		for _, p := range res {
			if p.Slug == "semen-portland-merah-putih" {
				found = true
			}
		}
		require.True(t, found, "expected 'semen' query to find semen-portland-merah-putih")

		res2, _, err := repo.Search(ctx, repository.SearchParams{Query: "portland 40"})
		require.NoError(t, err)
		found = false
		for _, p := range res2 {
			if p.Slug == "portland-cement-40kg" {
				found = true
			}
		}
		require.True(t, found, "expected 'portland 40' query to find portland-cement-40kg")
	})

	t.Run("AC-4.d excludes non-active product and suspended supplier", func(t *testing.T) {
		res, _, err := repo.Search(ctx, repository.SearchParams{Query: "Draft Product", Limit: 50})
		require.NoError(t, err)
		for _, p := range res {
			require.NotEqual(t, "draft-product", p.Slug)
		}

		res2, _, err := repo.Search(ctx, repository.SearchParams{Query: "Suspended Supplier Product", Limit: 50})
		require.NoError(t, err)
		for _, p := range res2 {
			require.NotEqual(t, "suspended-supplier-product", p.Slug)
		}
	})

	t.Run("price filter", func(t *testing.T) {
		res, _, err := repo.Search(ctx, repository.SearchParams{MinPrice: 60000, MaxPrice: 70000, Limit: 50})
		require.NoError(t, err)
		require.NotEmpty(t, res)
		for _, p := range res {
			require.GreaterOrEqual(t, p.PriceMin, 60000.0)
			require.LessOrEqual(t, p.PriceMax, 70000.0)
		}
	})

	t.Run("sort price_asc is ordered", func(t *testing.T) {
		res, _, err := repo.Search(ctx, repository.SearchParams{Sort: repository.SortPriceAsc, Limit: 50})
		require.NoError(t, err)
		for i := 1; i < len(res); i++ {
			require.LessOrEqual(t, res[i-1].PriceMin, res[i].PriceMin)
		}
	})

	t.Run("AC-4.e cursor pagination covers all active products without dupes or gaps", func(t *testing.T) {
		seen := map[uuid.UUID]bool{}
		cursor := ""
		pages := 0
		for {
			res, next, err := repo.Search(ctx, repository.SearchParams{Sort: repository.SortNewest, Cursor: cursor, Limit: 10})
			require.NoError(t, err)
			for _, p := range res {
				require.False(t, seen[p.ID], "duplicate product across pages: %s", p.Slug)
				seen[p.ID] = true
			}
			pages++
			require.Less(t, pages, 20, "pagination did not terminate")
			if next == "" {
				break
			}
			cursor = next
		}
		// 48 + 2 (portland/semen) active-approved products; draft + suspended excluded.
		require.Len(t, seen, 50)
	})
}
