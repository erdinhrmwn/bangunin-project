//go:build integration

// Package postgres_test — AC-4.c: stock adjustment guard + movement
// atomicity against a real Postgres transaction (not mocks).
package postgres_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/errs"

	postgresrepo "erdinhrmwn/bangunin/internal/repository/postgres"
)

// AC-4.c: adjusting below zero is rejected (409-mapped ErrConflict) and
// stock is left untouched; a valid +/- sequence produces two movement rows
// with correct stock_after each, applied atomically with the stock update.
func TestProductVariantRepository_AdjustStockWithMovement(t *testing.T) {
	pool := setupDB(t)
	f := seedBase(t, pool)
	productID := f.seedProduct(t, f.supplierID, "Semen Gresik 50kg", "semen-gresik-50kg", 60000, entity.ProductStatusActive)

	var variantID uuid.UUID
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT id FROM product_variants WHERE product_id = $1`, productID,
	).Scan(&variantID))
	// seedProduct seeds stock = 10.

	repo := postgresrepo.NewProductVariantRepository(pool)
	ctx := context.Background()

	_, err := repo.AdjustStockWithMovement(ctx, variantID, -20, &entity.StockMovement{
		ID: uuid.Must(uuid.NewV7()), VariantID: variantID,
		Type: entity.MovementTypeAdjustment, Qty: -20, RefType: entity.RefTypeManual,
	})
	require.ErrorIs(t, err, errs.ErrConflict)

	var stock int
	require.NoError(t, pool.QueryRow(ctx, `SELECT stock FROM product_variants WHERE id = $1`, variantID).Scan(&stock))
	require.Equal(t, 10, stock, "rejected adjustment must not touch stock")

	newStock, err := repo.AdjustStockWithMovement(ctx, variantID, 10, &entity.StockMovement{
		ID: uuid.Must(uuid.NewV7()), VariantID: variantID,
		Type: entity.MovementTypeAdjustment, Qty: 10, RefType: entity.RefTypeManual,
	})
	require.NoError(t, err)
	require.Equal(t, 20, newStock)

	newStock, err = repo.AdjustStockWithMovement(ctx, variantID, -3, &entity.StockMovement{
		ID: uuid.Must(uuid.NewV7()), VariantID: variantID,
		Type: entity.MovementTypeAdjustment, Qty: -3, RefType: entity.RefTypeManual,
	})
	require.NoError(t, err)
	require.Equal(t, 17, newStock)

	rows, err := pool.Query(ctx,
		`SELECT qty, stock_after FROM stock_movements WHERE variant_id = $1 ORDER BY created_at`, variantID)
	require.NoError(t, err)
	defer rows.Close()

	var got []struct{ Qty, StockAfter int }
	for rows.Next() {
		var qty, stockAfter int
		require.NoError(t, rows.Scan(&qty, &stockAfter))
		got = append(got, struct{ Qty, StockAfter int }{qty, stockAfter})
	}
	require.NoError(t, rows.Err())
	require.Len(t, got, 2, "rejected adjustment must not have written a movement row")
	require.Equal(t, 10, got[0].Qty)
	require.Equal(t, 20, got[0].StockAfter)
	require.Equal(t, -3, got[1].Qty)
	require.Equal(t, 17, got[1].StockAfter)
}
