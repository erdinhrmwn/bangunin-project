package repository

import (
	"context"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
)

// SupplierDocumentRepository — Upsert replaces any existing row for
// (supplier_id, doc_type); no upload history is kept.
type SupplierDocumentRepository interface {
	Upsert(ctx context.Context, d *entity.SupplierDocument) error
	FindBySupplierID(ctx context.Context, supplierID uuid.UUID) ([]*entity.SupplierDocument, error)
	FindBySupplierAndType(ctx context.Context, supplierID uuid.UUID, docType string) (*entity.SupplierDocument, error)
}
