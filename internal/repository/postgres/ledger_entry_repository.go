package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"erdinhrmwn/bangunin/internal/domain/entity"
)

type LedgerEntryRepository struct {
	db *pgxpool.Pool
}

func NewLedgerEntryRepository(db *pgxpool.Pool) *LedgerEntryRepository {
	return &LedgerEntryRepository{db: db}
}

func (r *LedgerEntryRepository) Create(ctx context.Context, e *entity.LedgerEntry) error {
	const q = `
		INSERT INTO ledger_entries (id, order_id, supplier_id, type, amount, balance_after)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at
	`
	err := r.db.QueryRow(ctx, q, e.ID, e.OrderID, e.SupplierID, e.Type, e.Amount, e.BalanceAfter).Scan(&e.CreatedAt)
	if err != nil {
		return fmt.Errorf("postgres: create ledger entry: %w", err)
	}
	return nil
}

func (r *LedgerEntryRepository) ListBySupplier(ctx context.Context, supplierID uuid.UUID, page, perPage int) ([]*entity.LedgerEntry, int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM ledger_entries WHERE supplier_id = $1`, supplierID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("postgres: count ledger entries: %w", err)
	}

	const q = selectLedgerEntryCols + `FROM ledger_entries WHERE supplier_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, q, supplierID, perPage, (page-1)*perPage)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres: list ledger entries: %w", err)
	}
	defer rows.Close()

	out, err := collectLedgerEntries(rows)
	return out, total, err
}

const selectLedgerEntryCols = `SELECT id, order_id, supplier_id, type, amount, balance_after, created_at `

func scanLedgerEntry(row rowScanner) (*entity.LedgerEntry, error) {
	var e entity.LedgerEntry
	err := row.Scan(&e.ID, &e.OrderID, &e.SupplierID, &e.Type, &e.Amount, &e.BalanceAfter, &e.CreatedAt)
	return &e, err
}

func collectLedgerEntries(rows pgx.Rows) ([]*entity.LedgerEntry, error) {
	var out []*entity.LedgerEntry
	for rows.Next() {
		e, err := scanLedgerEntry(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres: scan ledger entry: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
