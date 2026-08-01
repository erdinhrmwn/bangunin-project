package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/errs"
)

type WithdrawRequestRepository struct {
	db *pgxpool.Pool
}

func NewWithdrawRequestRepository(db *pgxpool.Pool) *WithdrawRequestRepository {
	return &WithdrawRequestRepository{db: db}
}

func (r *WithdrawRequestRepository) Create(ctx context.Context, w *entity.WithdrawRequest) error {
	const q = `
		INSERT INTO withdraw_requests (id, supplier_id, bank_account_id, amount, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING requested_at
	`
	err := r.db.QueryRow(ctx, q, w.ID, w.SupplierID, w.BankAccountID, w.Amount, w.Status).Scan(&w.RequestedAt)
	if err != nil {
		return fmt.Errorf("postgres: create withdraw request: %w", err)
	}
	return nil
}

func (r *WithdrawRequestRepository) Update(ctx context.Context, w *entity.WithdrawRequest) error {
	const q = `
		UPDATE withdraw_requests
		SET status = $2, disbursement_id = $3, processed_at = $4
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, q, w.ID, w.Status, w.DisbursementID, w.ProcessedAt)
	if err != nil {
		return fmt.Errorf("postgres: update withdraw request: %w", err)
	}
	return nil
}

func (r *WithdrawRequestRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.WithdrawRequest, error) {
	w, err := scanWithdrawRequest(r.db.QueryRow(ctx, selectWithdrawRequestCols+`FROM withdraw_requests WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errs.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: find withdraw request: %w", err)
	}
	return w, nil
}

func (r *WithdrawRequestRepository) ListBySupplier(ctx context.Context, supplierID uuid.UUID, page, perPage int) ([]*entity.WithdrawRequest, int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM withdraw_requests WHERE supplier_id = $1`, supplierID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("postgres: count withdraw requests: %w", err)
	}

	const q = selectWithdrawRequestCols + `FROM withdraw_requests WHERE supplier_id = $1 ORDER BY requested_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, q, supplierID, perPage, (page-1)*perPage)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres: list withdraw requests by supplier: %w", err)
	}
	defer rows.Close()

	out, err := collectWithdrawRequests(rows)
	return out, total, err
}

func (r *WithdrawRequestRepository) ListAll(ctx context.Context, status string, page, perPage int) ([]*entity.WithdrawRequest, int, error) {
	where := "true"
	if status != "" {
		where = "status = $1"
	}

	countQ := fmt.Sprintf(`SELECT count(*) FROM withdraw_requests WHERE %s`, where)
	var total int
	var err error
	if status != "" {
		err = r.db.QueryRow(ctx, countQ, status).Scan(&total)
	} else {
		err = r.db.QueryRow(ctx, countQ).Scan(&total)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("postgres: count withdraw requests: %w", err)
	}

	var rows pgx.Rows
	if status != "" {
		q := selectWithdrawRequestCols + fmt.Sprintf(`FROM withdraw_requests WHERE %s ORDER BY requested_at DESC LIMIT $2 OFFSET $3`, where)
		rows, err = r.db.Query(ctx, q, status, perPage, (page-1)*perPage)
	} else {
		q := selectWithdrawRequestCols + fmt.Sprintf(`FROM withdraw_requests WHERE %s ORDER BY requested_at DESC LIMIT $1 OFFSET $2`, where)
		rows, err = r.db.Query(ctx, q, perPage, (page-1)*perPage)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("postgres: list withdraw requests: %w", err)
	}
	defer rows.Close()

	out, err := collectWithdrawRequests(rows)
	return out, total, err
}

const selectWithdrawRequestCols = `
	SELECT id, supplier_id, bank_account_id, amount, status, disbursement_id, requested_at, processed_at
`

func scanWithdrawRequest(row rowScanner) (*entity.WithdrawRequest, error) {
	var w entity.WithdrawRequest
	err := row.Scan(&w.ID, &w.SupplierID, &w.BankAccountID, &w.Amount, &w.Status, &w.DisbursementID, &w.RequestedAt, &w.ProcessedAt)
	return &w, err
}

func collectWithdrawRequests(rows pgx.Rows) ([]*entity.WithdrawRequest, error) {
	var out []*entity.WithdrawRequest
	for rows.Next() {
		w, err := scanWithdrawRequest(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres: scan withdraw request: %w", err)
		}
		out = append(out, w)
	}
	return out, rows.Err()
}
