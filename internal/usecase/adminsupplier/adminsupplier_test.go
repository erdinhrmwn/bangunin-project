package adminsupplier_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/repository/mocks"
	svcmocks "erdinhrmwn/bangunin/internal/domain/service/mocks"
	adminsupplierusecase "erdinhrmwn/bangunin/internal/usecase/adminsupplier"
	"erdinhrmwn/bangunin/pkg/apperr"
)

func newUsecase(t *testing.T) (*adminsupplierusecase.Usecase, *mocks.MockSupplierRepository, *mocks.MockSupplierDocumentRepository, *mocks.MockUserRepository, *mocks.MockAuditLogRepository, *mocks.MockNotificationRepository, *svcmocks.MockEmailEnqueuer) {
	t.Helper()
	suppliers := mocks.NewMockSupplierRepository(t)
	docs := mocks.NewMockSupplierDocumentRepository(t)
	users := mocks.NewMockUserRepository(t)
	audit := mocks.NewMockAuditLogRepository(t)
	notify := mocks.NewMockNotificationRepository(t)
	email := svcmocks.NewMockEmailEnqueuer(t)
	uc := adminsupplierusecase.New(suppliers, docs, users, audit, notify, email)
	return uc, suppliers, docs, users, audit, notify, email
}

func mustUUID(t *testing.T) uuid.UUID {
	t.Helper()
	id, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func appErrCode(t *testing.T, err error) string {
	t.Helper()
	return apperr.From(err).Code
}

func TestApprove_NotPending_Conflict(t *testing.T) {
	uc, suppliers, _, _, _, _, _ := newUsecase(t)
	id := mustUUID(t)
	s := &entity.Supplier{ID: id, Status: entity.SupplierStatusApproved}
	suppliers.EXPECT().FindByID(mock.Anything, id).Return(s, nil)

	_, err := uc.Approve(context.Background(), mustUUID(t), id, "127.0.0.1")

	assert.Equal(t, "INVALID_STATUS", appErrCode(t, err))
}

func TestApprove_Success(t *testing.T) {
	uc, suppliers, _, users, audit, notify, email := newUsecase(t)
	id := mustUUID(t)
	userID := mustUUID(t)
	s := &entity.Supplier{ID: id, UserID: userID, Status: entity.SupplierStatusPending}
	suppliers.EXPECT().FindByID(mock.Anything, id).Return(s, nil)
	suppliers.EXPECT().Update(mock.Anything, mock.MatchedBy(func(s *entity.Supplier) bool {
		return s.Status == entity.SupplierStatusApproved && s.VerifiedAt != nil
	})).Return(nil)
	audit.EXPECT().Create(mock.Anything, mock.MatchedBy(func(a *entity.AuditLog) bool {
		return a.Action == "approve" && a.EntityID == id
	})).Return(nil)
	notify.EXPECT().Create(mock.Anything, mock.MatchedBy(func(n *entity.Notification) bool {
		return n.UserID == userID && n.Type == entity.NotificationTypeApproval
	})).Return(nil)
	users.EXPECT().FindByID(mock.Anything, userID).Return(&entity.User{ID: userID, Email: "a@example.com"}, nil)
	email.EXPECT().EnqueueEmail("a@example.com", mock.Anything, mock.Anything).Return(nil)

	got, err := uc.Approve(context.Background(), mustUUID(t), id, "127.0.0.1")

	assert.NoError(t, err)
	assert.Equal(t, entity.SupplierStatusApproved, got.Status)
}

func TestReject_ResetsToDraft(t *testing.T) {
	uc, suppliers, _, users, audit, notify, email := newUsecase(t)
	id := mustUUID(t)
	userID := mustUUID(t)
	s := &entity.Supplier{ID: id, UserID: userID, Status: entity.SupplierStatusPending}
	suppliers.EXPECT().FindByID(mock.Anything, id).Return(s, nil)
	suppliers.EXPECT().Update(mock.Anything, mock.MatchedBy(func(s *entity.Supplier) bool {
		return s.Status == entity.SupplierStatusDraft
	})).Return(nil)
	audit.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)
	notify.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)
	users.EXPECT().FindByID(mock.Anything, userID).Return(&entity.User{ID: userID, Email: "a@example.com"}, nil)
	email.EXPECT().EnqueueEmail(mock.Anything, mock.Anything, mock.Anything).Return(nil)

	got, err := uc.Reject(context.Background(), mustUUID(t), id, "missing NIB", "127.0.0.1")

	assert.NoError(t, err)
	assert.Equal(t, entity.SupplierStatusDraft, got.Status)
}

func TestSuspend_NotApproved_Conflict(t *testing.T) {
	uc, suppliers, _, _, _, _, _ := newUsecase(t)
	id := mustUUID(t)
	s := &entity.Supplier{ID: id, Status: entity.SupplierStatusDraft}
	suppliers.EXPECT().FindByID(mock.Anything, id).Return(s, nil)

	_, err := uc.Suspend(context.Background(), mustUUID(t), id, "reason", "127.0.0.1")

	assert.Equal(t, "INVALID_STATUS", appErrCode(t, err))
}

func TestGet_ReturnsDetailWithDocuments(t *testing.T) {
	uc, suppliers, docs, _, _, _, _ := newUsecase(t)
	id := mustUUID(t)
	s := &entity.Supplier{ID: id}
	suppliers.EXPECT().FindByID(mock.Anything, id).Return(s, nil)
	docList := []*entity.SupplierDocument{{ID: mustUUID(t), SupplierID: id}}
	docs.EXPECT().FindBySupplierID(mock.Anything, id).Return(docList, nil)

	detail, err := uc.Get(context.Background(), id)

	assert.NoError(t, err)
	assert.Len(t, detail.Documents, 1)
}
