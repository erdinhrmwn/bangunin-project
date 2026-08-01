package supplier_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/errs"
	"erdinhrmwn/bangunin/internal/domain/repository/mocks"
	supplierusecase "erdinhrmwn/bangunin/internal/usecase/supplier"
	"erdinhrmwn/bangunin/pkg/apperr"
)

func newUsecase(t *testing.T) (*supplierusecase.Usecase, *mocks.MockSupplierRepository, *mocks.MockSupplierDocumentRepository, *mocks.MockSupplierBankAccountRepository) {
	t.Helper()
	suppliers := mocks.NewMockSupplierRepository(t)
	docs := mocks.NewMockSupplierDocumentRepository(t)
	banks := mocks.NewMockSupplierBankAccountRepository(t)
	return supplierusecase.New(suppliers, docs, banks), suppliers, docs, banks
}

func appErrCode(t *testing.T, err error) string {
	t.Helper()
	return apperr.From(err).Code
}

func TestCreateProfile_AlreadyExists(t *testing.T) {
	uc, suppliers, _, _ := newUsecase(t)
	userID := uuid.Must(uuid.NewV7())
	suppliers.EXPECT().FindByUserID(mock.Anything, userID).
		Return(&entity.Supplier{ID: uuid.Must(uuid.NewV7())}, nil)

	_, err := uc.CreateProfile(context.Background(), userID, supplierusecase.ProfileInput{StoreName: "Toko A"})

	assert.Equal(t, "SUPPLIER_EXISTS", appErrCode(t, err))
}

func TestCreateProfile_Success(t *testing.T) {
	uc, suppliers, _, _ := newUsecase(t)
	userID := uuid.Must(uuid.NewV7())
	suppliers.EXPECT().FindByUserID(mock.Anything, userID).Return(nil, errs.ErrNotFound)
	suppliers.EXPECT().FindBySlug(mock.Anything, "toko-a").Return(nil, errs.ErrNotFound)
	suppliers.EXPECT().Create(mock.Anything, mock.MatchedBy(func(s *entity.Supplier) bool {
		return s.Slug == "toko-a" && s.Status == entity.SupplierStatusDraft
	})).Return(nil)

	s, err := uc.CreateProfile(context.Background(), userID, supplierusecase.ProfileInput{StoreName: "Toko A"})

	assert.NoError(t, err)
	assert.Equal(t, "toko-a", s.Slug)
}

func TestCreateProfile_SlugCollision(t *testing.T) {
	uc, suppliers, _, _ := newUsecase(t)
	userID := uuid.Must(uuid.NewV7())
	other := uuid.Must(uuid.NewV7())
	suppliers.EXPECT().FindByUserID(mock.Anything, userID).Return(nil, errs.ErrNotFound)
	suppliers.EXPECT().FindBySlug(mock.Anything, "toko-a").Return(&entity.Supplier{ID: other}, nil)
	suppliers.EXPECT().FindBySlug(mock.Anything, "toko-a-2").Return(nil, errs.ErrNotFound)
	suppliers.EXPECT().Create(mock.Anything, mock.MatchedBy(func(s *entity.Supplier) bool {
		return s.Slug == "toko-a-2"
	})).Return(nil)

	s, err := uc.CreateProfile(context.Background(), userID, supplierusecase.ProfileInput{StoreName: "Toko A"})

	assert.NoError(t, err)
	assert.Equal(t, "toko-a-2", s.Slug)
}

func TestUploadDocument_InvalidType(t *testing.T) {
	uc, _, _, _ := newUsecase(t)
	_, err := uc.UploadDocument(context.Background(), uuid.Must(uuid.NewV7()), "invalid", "key")
	assert.Equal(t, "VALIDATION_ERROR", appErrCode(t, err))
}

func TestUploadDocument_Success(t *testing.T) {
	uc, suppliers, docs, _ := newUsecase(t)
	userID := uuid.Must(uuid.NewV7())
	supplierID := uuid.Must(uuid.NewV7())
	suppliers.EXPECT().FindByUserID(mock.Anything, userID).Return(&entity.Supplier{ID: supplierID}, nil)
	docs.EXPECT().Upsert(mock.Anything, mock.MatchedBy(func(d *entity.SupplierDocument) bool {
		return d.SupplierID == supplierID && d.DocType == entity.DocTypeNIB && d.Status == entity.DocStatusPending
	})).Return(nil)

	d, err := uc.UploadDocument(context.Background(), userID, entity.DocTypeNIB, "key")

	assert.NoError(t, err)
	assert.Equal(t, entity.DocTypeNIB, d.DocType)
}

func TestSubmit_MissingDocuments(t *testing.T) {
	uc, suppliers, docs, _ := newUsecase(t)
	userID := uuid.Must(uuid.NewV7())
	supplierID := uuid.Must(uuid.NewV7())
	suppliers.EXPECT().FindByUserID(mock.Anything, userID).Return(&entity.Supplier{
		ID: supplierID, Status: entity.SupplierStatusDraft, StoreName: "A", OriginCityID: 1, PickupAddress: "addr",
	}, nil)
	docs.EXPECT().FindBySupplierID(mock.Anything, supplierID).Return(nil, nil)

	err := uc.Submit(context.Background(), userID)

	appErr := apperr.From(err)
	assert.Equal(t, "VALIDATION_ERROR", appErr.Code)
	assert.Contains(t, appErr.Message, "ktp")
	assert.Contains(t, appErr.Message, "nib")
}

func TestSubmit_AlreadySubmitted(t *testing.T) {
	uc, suppliers, _, _ := newUsecase(t)
	userID := uuid.Must(uuid.NewV7())
	suppliers.EXPECT().FindByUserID(mock.Anything, userID).Return(&entity.Supplier{
		Status: entity.SupplierStatusPending,
	}, nil)

	err := uc.Submit(context.Background(), userID)

	assert.Equal(t, "INVALID_STATUS", appErrCode(t, err))
}

func TestSubmit_Success(t *testing.T) {
	uc, suppliers, docs, _ := newUsecase(t)
	userID := uuid.Must(uuid.NewV7())
	supplierID := uuid.Must(uuid.NewV7())
	s := &entity.Supplier{
		ID: supplierID, Status: entity.SupplierStatusDraft, StoreName: "A", OriginCityID: 1, PickupAddress: "addr",
	}
	suppliers.EXPECT().FindByUserID(mock.Anything, userID).Return(s, nil)
	docs.EXPECT().FindBySupplierID(mock.Anything, supplierID).Return([]*entity.SupplierDocument{
		{DocType: entity.DocTypeKTP}, {DocType: entity.DocTypeNIB},
	}, nil)
	suppliers.EXPECT().Update(mock.Anything, mock.MatchedBy(func(s *entity.Supplier) bool {
		return s.Status == entity.SupplierStatusPending
	})).Return(nil)

	err := uc.Submit(context.Background(), userID)

	assert.NoError(t, err)
}

func TestUpsertBankAccount_UnsetsPreviousDefault(t *testing.T) {
	uc, suppliers, _, banks := newUsecase(t)
	userID := uuid.Must(uuid.NewV7())
	supplierID := uuid.Must(uuid.NewV7())
	suppliers.EXPECT().FindByUserID(mock.Anything, userID).Return(&entity.Supplier{ID: supplierID}, nil)
	banks.EXPECT().UnsetDefault(mock.Anything, supplierID).Return(nil)
	banks.EXPECT().Create(mock.Anything, mock.MatchedBy(func(a *entity.SupplierBankAccount) bool {
		return a.SupplierID == supplierID && a.IsDefault
	})).Return(nil)

	_, err := uc.UpsertBankAccount(context.Background(), userID, nil, supplierusecase.BankAccountInput{
		BankCode: "BCA", AccountNumber: "1234567890", AccountName: "A", IsDefault: true,
	})

	assert.NoError(t, err)
}

func TestDeleteBankAccount_NotOwner(t *testing.T) {
	uc, suppliers, _, banks := newUsecase(t)
	userID := uuid.Must(uuid.NewV7())
	supplierID := uuid.Must(uuid.NewV7())
	accountID := uuid.Must(uuid.NewV7())
	suppliers.EXPECT().FindByUserID(mock.Anything, userID).Return(&entity.Supplier{ID: supplierID}, nil)
	banks.EXPECT().FindByID(mock.Anything, accountID).Return(&entity.SupplierBankAccount{
		ID: accountID, SupplierID: uuid.Must(uuid.NewV7()),
	}, nil)

	err := uc.DeleteBankAccount(context.Background(), userID, accountID)

	assert.ErrorIs(t, err, errs.ErrNotFound)
}
