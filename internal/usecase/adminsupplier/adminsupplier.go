// Package adminsupplier implements admin review of supplier applications:
// list/detail, approve/reject/suspend with audit trail + notification
// (FR-3.6–FR-3.8).
package adminsupplier

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/repository"
	"erdinhrmwn/bangunin/internal/domain/service"
	"erdinhrmwn/bangunin/internal/pkg/notify"
	"erdinhrmwn/bangunin/pkg/apperr"
)

const auditEntitySupplier = "supplier"

type Usecase struct {
	suppliers repository.SupplierRepository
	docs      repository.SupplierDocumentRepository
	users     repository.UserRepository
	audit     repository.AuditLogRepository
	notify    repository.NotificationRepository
	email     service.EmailEnqueuer
}

func New(suppliers repository.SupplierRepository, docs repository.SupplierDocumentRepository, users repository.UserRepository, audit repository.AuditLogRepository, notify repository.NotificationRepository, email service.EmailEnqueuer) *Usecase {
	return &Usecase{suppliers: suppliers, docs: docs, users: users, audit: audit, notify: notify, email: email}
}

func (u *Usecase) List(ctx context.Context, status, q string, page, perPage int) ([]*entity.Supplier, int, error) {
	return u.suppliers.List(ctx, status, q, page, perPage)
}

// AuditLogs returns admin action history (FR-3.7).
func (u *Usecase) AuditLogs(ctx context.Context, from, to time.Time, page, perPage int) ([]*entity.AuditLog, int, error) {
	return u.audit.List(ctx, from, to, page, perPage)
}

type SupplierDetail struct {
	*entity.Supplier
	Documents []*entity.SupplierDocument
}

// Get returns the supplier plus its documents (FR-3.6). Document FileURL is
// already the direct object URL from upload — no separate presign step.
func (u *Usecase) Get(ctx context.Context, id uuid.UUID) (*SupplierDetail, error) {
	s, err := u.suppliers.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	docs, err := u.docs.FindBySupplierID(ctx, s.ID)
	if err != nil {
		return nil, err
	}
	return &SupplierDetail{Supplier: s, Documents: docs}, nil
}

// Approve moves a pending supplier to approved and sets VerifiedAt (FR-3.6).
func (u *Usecase) Approve(ctx context.Context, actorID, id uuid.UUID, ip string) (*entity.Supplier, error) {
	s, err := u.suppliers.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if s.Status != entity.SupplierStatusPending {
		return nil, apperr.New("INVALID_STATUS", "Supplier is not pending review", 409)
	}

	before := s.Status
	now := time.Now()
	s.Status = entity.SupplierStatusApproved
	s.VerifiedAt = &now
	if err := u.suppliers.Update(ctx, s); err != nil {
		return nil, err
	}

	if err := u.recordAndNotify(ctx, actorID, s, "approve", ip, "", before); err != nil {
		return nil, err
	}
	return s, nil
}

// Reject sends a pending supplier back to draft with a reason so they can
// fix and resubmit (FR-3.6).
func (u *Usecase) Reject(ctx context.Context, actorID, id uuid.UUID, reason, ip string) (*entity.Supplier, error) {
	s, err := u.suppliers.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if s.Status != entity.SupplierStatusPending {
		return nil, apperr.New("INVALID_STATUS", "Supplier is not pending review", 409)
	}

	before := s.Status
	s.Status = entity.SupplierStatusDraft
	if err := u.suppliers.Update(ctx, s); err != nil {
		return nil, err
	}

	if err := u.recordAndNotify(ctx, actorID, s, "reject", ip, reason, before); err != nil {
		return nil, err
	}
	return s, nil
}

// Suspend deactivates an approved supplier (FR-3.6).
func (u *Usecase) Suspend(ctx context.Context, actorID, id uuid.UUID, reason, ip string) (*entity.Supplier, error) {
	s, err := u.suppliers.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if s.Status != entity.SupplierStatusApproved {
		return nil, apperr.New("INVALID_STATUS", "Only approved suppliers can be suspended", 409)
	}

	before := s.Status
	s.Status = entity.SupplierStatusSuspended
	if err := u.suppliers.Update(ctx, s); err != nil {
		return nil, err
	}

	if err := u.recordAndNotify(ctx, actorID, s, "suspend", ip, reason, before); err != nil {
		return nil, err
	}
	return s, nil
}

func (u *Usecase) recordAndNotify(ctx context.Context, actorID uuid.UUID, s *entity.Supplier, action, ip, reason, before string) error {
	auditID, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("adminsupplier: generate audit id: %w", err)
	}
	if err := u.audit.Create(ctx, &entity.AuditLog{
		ID:         auditID,
		ActorID:    actorID,
		Action:     action,
		EntityType: auditEntitySupplier,
		EntityID:   s.ID,
		Metadata:   map[string]any{"reason": reason, "before": before, "after": s.Status},
		IPAddress:  ip,
	}); err != nil {
		return err
	}

	title, body := notificationText(action, reason)
	return notify.SendWithEmail(ctx, u.notify, u.users, u.email, s.UserID, entity.NotificationTypeApproval, title, body)
}

func notificationText(action, reason string) (title, body string) {
	switch action {
	case "approve":
		return "Supplier application approved", "Your supplier account has been approved."
	case "reject":
		return "Supplier application rejected", "Your supplier application was rejected: " + reason
	default:
		return "Supplier account suspended", "Your supplier account has been suspended: " + reason
	}
}
