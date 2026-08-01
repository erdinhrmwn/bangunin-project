package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"erdinhrmwn/bangunin/internal/domain/entity"
)

type SupplierDocumentRepository struct {
	db *pgxpool.Pool
}

func NewSupplierDocumentRepository(db *pgxpool.Pool) *SupplierDocumentRepository {
	return &SupplierDocumentRepository{db: db}
}

func (r *SupplierDocumentRepository) Upsert(ctx context.Context, d *entity.SupplierDocument) error {
	const q = `
		INSERT INTO supplier_documents (id, supplier_id, doc_type, file_url, status, rejection_note)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (supplier_id, doc_type)
		DO UPDATE SET file_url = $4, status = $5, rejection_note = $6
	`
	_, err := r.db.Exec(ctx, q, d.ID, d.SupplierID, d.DocType, d.FileURL, d.Status, d.RejectionNote)
	if err != nil {
		return fmt.Errorf("postgres: upsert supplier document: %w", err)
	}
	return nil
}

func (r *SupplierDocumentRepository) FindBySupplierID(ctx context.Context, supplierID uuid.UUID) ([]*entity.SupplierDocument, error) {
	const q = `
		SELECT id, supplier_id, doc_type, file_url, status, rejection_note, created_at
		FROM supplier_documents WHERE supplier_id = $1
	`
	rows, err := r.db.Query(ctx, q, supplierID)
	if err != nil {
		return nil, fmt.Errorf("postgres: list supplier documents: %w", err)
	}
	defer rows.Close()

	var out []*entity.SupplierDocument
	for rows.Next() {
		var d entity.SupplierDocument
		if err := rows.Scan(&d.ID, &d.SupplierID, &d.DocType, &d.FileURL, &d.Status, &d.RejectionNote, &d.CreatedAt); err != nil {
			return nil, fmt.Errorf("postgres: scan supplier document: %w", err)
		}
		out = append(out, &d)
	}
	return out, rows.Err()
}

func (r *SupplierDocumentRepository) FindBySupplierAndType(ctx context.Context, supplierID uuid.UUID, docType string) (*entity.SupplierDocument, error) {
	const q = `
		SELECT id, supplier_id, doc_type, file_url, status, rejection_note, created_at
		FROM supplier_documents WHERE supplier_id = $1 AND doc_type = $2
	`
	var d entity.SupplierDocument
	err := r.db.QueryRow(ctx, q, supplierID, docType).Scan(
		&d.ID, &d.SupplierID, &d.DocType, &d.FileURL, &d.Status, &d.RejectionNote, &d.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: find supplier document: %w", err)
	}
	return &d, nil
}
