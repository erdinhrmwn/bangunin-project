package user_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/repository/mocks"
	userusecase "erdinhrmwn/bangunin/internal/usecase/user"
	"erdinhrmwn/bangunin/pkg/apperr"
	"erdinhrmwn/bangunin/pkg/hash"
)

func newUsecase(t *testing.T) (*userusecase.Usecase, *mocks.MockUserRepository) {
	t.Helper()
	users := mocks.NewMockUserRepository(t)
	return userusecase.New(users), users
}

// ChangePassword: wrong old password -> 422 INVALID_PASSWORD, no update call.
func TestChangePassword_WrongOldPassword(t *testing.T) {
	uc, users := newUsecase(t)
	id := mustUUID(t)
	hashed, err := hash.Hash("correct-pass1")
	assert.NoError(t, err)
	usr := &entity.User{ID: id, PasswordHash: hashed}
	users.EXPECT().FindByID(mock.Anything, id).Return(usr, nil)

	err = uc.ChangePassword(context.Background(), id, "wrong-pass", "new-pass1")

	assert.Equal(t, "INVALID_PASSWORD", apperr.From(err).Code)
}

// ChangePassword: correct old password -> hashes new password and persists.
func TestChangePassword_Success(t *testing.T) {
	uc, users := newUsecase(t)
	id := mustUUID(t)
	hashed, err := hash.Hash("correct-pass1")
	assert.NoError(t, err)
	usr := &entity.User{ID: id, PasswordHash: hashed}
	users.EXPECT().FindByID(mock.Anything, id).Return(usr, nil)
	users.EXPECT().Update(mock.Anything, usr).Return(nil)

	err = uc.ChangePassword(context.Background(), id, "correct-pass1", "new-pass1")

	assert.NoError(t, err)
}

func mustUUID(t *testing.T) uuid.UUID {
	t.Helper()
	u, err := uuid.NewV7()
	assert.NoError(t, err)
	return u
}
