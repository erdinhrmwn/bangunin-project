// Package supplier implements store profile, document, bank account, and
// submission operations for a supplier's own account (FR-3.2–FR-3.5).
package supplier

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/errs"
	"erdinhrmwn/bangunin/internal/domain/repository"
	"erdinhrmwn/bangunin/pkg/apperr"
)

type Usecase struct {
	suppliers repository.SupplierRepository
	docs      repository.SupplierDocumentRepository
	banks     repository.SupplierBankAccountRepository
}

func New(suppliers repository.SupplierRepository, docs repository.SupplierDocumentRepository, banks repository.SupplierBankAccountRepository) *Usecase {
	return &Usecase{suppliers: suppliers, docs: docs, banks: banks}
}

type ProfileInput struct {
	StoreName       string
	Description     string
	OriginCityID    int
	PickupAddress   string
	OwnFleetEnabled bool
	FleetCoverageKM int
	FleetFlatRate   float64
}

// CreateProfile creates a supplier's store profile (FR-3.2). One user may
// have at most one supplier.
func (u *Usecase) CreateProfile(ctx context.Context, userID uuid.UUID, in ProfileInput) (*entity.Supplier, error) {
	if _, err := u.suppliers.FindByUserID(ctx, userID); err == nil {
		return nil, apperr.New("SUPPLIER_EXISTS", "Supplier profile already exists", 409)
	} else if !errors.Is(err, errs.ErrNotFound) {
		return nil, err
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("supplier: generate id: %w", err)
	}
	slug, err := u.uniqueSlug(ctx, in.StoreName, id)
	if err != nil {
		return nil, err
	}

	s := &entity.Supplier{
		ID:              id,
		UserID:          userID,
		StoreName:       in.StoreName,
		Slug:            slug,
		Description:     in.Description,
		Status:          entity.SupplierStatusDraft,
		OriginCityID:    in.OriginCityID,
		PickupAddress:   in.PickupAddress,
		OwnFleetEnabled: in.OwnFleetEnabled,
		FleetCoverageKM: in.FleetCoverageKM,
		FleetFlatRate:   in.FleetFlatRate,
	}
	if err := u.suppliers.Create(ctx, s); err != nil {
		return nil, err
	}
	return s, nil
}

// UpdateProfile updates the caller's store profile (FR-3.2).
func (u *Usecase) UpdateProfile(ctx context.Context, userID uuid.UUID, in ProfileInput) (*entity.Supplier, error) {
	s, err := u.suppliers.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if in.StoreName != s.StoreName {
		slug, err := u.uniqueSlug(ctx, in.StoreName, s.ID)
		if err != nil {
			return nil, err
		}
		s.Slug = slug
	}
	s.StoreName = in.StoreName
	s.Description = in.Description
	s.OriginCityID = in.OriginCityID
	s.PickupAddress = in.PickupAddress
	s.OwnFleetEnabled = in.OwnFleetEnabled
	s.FleetCoverageKM = in.FleetCoverageKM
	s.FleetFlatRate = in.FleetFlatRate

	if err := u.suppliers.Update(ctx, s); err != nil {
		return nil, err
	}
	return s, nil
}

func (u *Usecase) Profile(ctx context.Context, userID uuid.UUID) (*entity.Supplier, error) {
	return u.suppliers.FindByUserID(ctx, userID)
}

var validDocTypes = map[string]bool{
	entity.DocTypeNIB:  true,
	entity.DocTypeKTP:  true,
	entity.DocTypeNPWP: true,
}

// UploadDocument records an uploaded document's storage key (FR-3.3).
// Re-uploading a doc_type replaces the existing row and resets it to
// pending review.
func (u *Usecase) UploadDocument(ctx context.Context, userID uuid.UUID, docType, fileKey string) (*entity.SupplierDocument, error) {
	if !validDocTypes[docType] {
		return nil, apperr.New("VALIDATION_ERROR", "Invalid doc_type", 422)
	}
	s, err := u.suppliers.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("supplier: generate id: %w", err)
	}
	d := &entity.SupplierDocument{
		ID:         id,
		SupplierID: s.ID,
		DocType:    docType,
		FileURL:    fileKey,
		Status:     entity.DocStatusPending,
	}
	if err := u.docs.Upsert(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

func (u *Usecase) ListDocuments(ctx context.Context, userID uuid.UUID) ([]*entity.SupplierDocument, error) {
	s, err := u.suppliers.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return u.docs.FindBySupplierID(ctx, s.ID)
}

type BankAccountInput struct {
	BankCode      string
	AccountNumber string
	AccountName   string
	IsDefault     bool
}

// UpsertBankAccount creates or updates a bank account (FR-3.4). When
// IsDefault is set, every other account for the supplier is unset first so
// exactly one stays default.
func (u *Usecase) UpsertBankAccount(ctx context.Context, userID uuid.UUID, id *uuid.UUID, in BankAccountInput) (*entity.SupplierBankAccount, error) {
	s, err := u.suppliers.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	a := &entity.SupplierBankAccount{
		SupplierID:    s.ID,
		BankCode:      in.BankCode,
		AccountNumber: in.AccountNumber,
		AccountName:   in.AccountName,
		IsDefault:     in.IsDefault,
	}

	if id != nil {
		existing, err := u.banks.FindByID(ctx, *id)
		if err != nil {
			return nil, err
		}
		if existing.SupplierID != s.ID {
			return nil, errs.ErrNotFound
		}
		a.ID = *id
	} else {
		newID, err := uuid.NewV7()
		if err != nil {
			return nil, fmt.Errorf("supplier: generate id: %w", err)
		}
		a.ID = newID
	}

	if in.IsDefault {
		if err := u.banks.UnsetDefault(ctx, s.ID); err != nil {
			return nil, err
		}
	}

	if id != nil {
		err = u.banks.Update(ctx, a)
	} else {
		err = u.banks.Create(ctx, a)
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (u *Usecase) ListBankAccounts(ctx context.Context, userID uuid.UUID) ([]*entity.SupplierBankAccount, error) {
	s, err := u.suppliers.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return u.banks.FindBySupplierID(ctx, s.ID)
}

func (u *Usecase) DeleteBankAccount(ctx context.Context, userID, id uuid.UUID) error {
	s, err := u.suppliers.FindByUserID(ctx, userID)
	if err != nil {
		return err
	}
	a, err := u.banks.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if a.SupplierID != s.ID {
		return errs.ErrNotFound
	}
	return u.banks.Delete(ctx, id)
}

// Submit moves the supplier from draft to pending review (FR-3.5). Requires
// a complete profile plus KTP and NIB documents on file.
func (u *Usecase) Submit(ctx context.Context, userID uuid.UUID) error {
	s, err := u.suppliers.FindByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if s.Status != entity.SupplierStatusDraft {
		return apperr.New("INVALID_STATUS", "Supplier is already submitted", 409)
	}

	var missing []string
	if s.StoreName == "" {
		missing = append(missing, "store_name")
	}
	if s.OriginCityID <= 0 {
		missing = append(missing, "origin_city_id")
	}
	if s.PickupAddress == "" {
		missing = append(missing, "pickup_address")
	}

	docs, err := u.docs.FindBySupplierID(ctx, s.ID)
	if err != nil {
		return err
	}
	has := map[string]bool{}
	for _, d := range docs {
		has[d.DocType] = true
	}
	if !has[entity.DocTypeKTP] {
		missing = append(missing, "ktp")
	}
	if !has[entity.DocTypeNIB] {
		missing = append(missing, "nib")
	}

	if len(missing) > 0 {
		return apperr.New("VALIDATION_ERROR", "Missing required fields/documents: "+strings.Join(missing, ", "), 422)
	}

	s.Status = entity.SupplierStatusPending
	return u.suppliers.Update(ctx, s)
}

var slugNonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(name string) string {
	s := slugNonAlnum.ReplaceAllString(strings.ToLower(name), "-")
	return strings.Trim(s, "-")
}

// uniqueSlug slugifies name and appends a short suffix on collision,
// excluding the supplier identified by selfID.
func (u *Usecase) uniqueSlug(ctx context.Context, name string, selfID uuid.UUID) (string, error) {
	base := slugify(name)
	slug := base
	for i := 2; ; i++ {
		existing, err := u.suppliers.FindBySlug(ctx, slug)
		if errors.Is(err, errs.ErrNotFound) {
			return slug, nil
		}
		if err != nil {
			return "", err
		}
		if existing.ID == selfID {
			return slug, nil
		}
		slug = fmt.Sprintf("%s-%d", base, i)
	}
}
