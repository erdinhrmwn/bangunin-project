package notification_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/repository/mocks"

	notificationusecase "erdinhrmwn/bangunin/internal/usecase/notification"
)

func newUsecase(t *testing.T) (*notificationusecase.Usecase, *mocks.MockNotificationRepository) {
	t.Helper()
	notify := mocks.NewMockNotificationRepository(t)
	return notificationusecase.New(notify), notify
}

func mustUUID(t *testing.T) uuid.UUID {
	t.Helper()
	id, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestList_ReturnsFromRepo(t *testing.T) {
	uc, notify := newUsecase(t)
	userID := mustUUID(t)
	want := []*entity.Notification{{ID: mustUUID(t), UserID: userID}}
	notify.EXPECT().FindByUserID(mock.Anything, userID, 1, 20).Return(want, 1, nil)

	got, total, err := uc.List(context.Background(), userID, 1, 20)

	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, want, got)
}

func TestMarkRead_DelegatesToRepo(t *testing.T) {
	uc, notify := newUsecase(t)
	id, userID := mustUUID(t), mustUUID(t)
	notify.EXPECT().MarkRead(mock.Anything, id, userID).Return(nil)

	err := uc.MarkRead(context.Background(), id, userID)

	assert.NoError(t, err)
}
