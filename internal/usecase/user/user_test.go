package user_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/repository/mocks"
	"erdinhrmwn/bangunin/pkg/apperr"
	"erdinhrmwn/bangunin/pkg/hash"

	userusecase "erdinhrmwn/bangunin/internal/usecase/user"
)

func newUsecase(t *testing.T) (*userusecase.Usecase, *mocks.MockUserRepository) {
	t.Helper()
	uc, users, _ := newUsecaseWithAddresses(t)
	return uc, users
}

func newUsecaseWithAddresses(t *testing.T) (*userusecase.Usecase, *mocks.MockUserRepository, *mocks.MockUserAddressRepository) {
	t.Helper()
	users := mocks.NewMockUserRepository(t)
	addresses := mocks.NewMockUserAddressRepository(t)
	return userusecase.New(users, addresses), users, addresses
}

// CreateAddress: persists and, when default, clears other defaults.
func TestCreateAddress_Default(t *testing.T) {
	uc, _, addresses := newUsecaseWithAddresses(t)
	userID := mustUUID(t)
	addresses.EXPECT().Create(mock.Anything, mock.AnythingOfType("*entity.UserAddress")).Return(nil)
	addresses.EXPECT().ClearDefault(mock.Anything, userID, mock.AnythingOfType("uuid.UUID")).Return(nil)

	a, err := uc.CreateAddress(context.Background(), userID, userusecase.AddressInput{Label: "Home", IsDefault: true})

	assert.NoError(t, err)
	assert.Equal(t, userID, a.UserID)
}

// UpdateAddress: address owned by a different user -> 403 FORBIDDEN, no update call.
func TestUpdateAddress_NotOwner(t *testing.T) {
	uc, _, addresses := newUsecaseWithAddresses(t)
	id, userID, otherUserID := mustUUID(t), mustUUID(t), mustUUID(t)
	addresses.EXPECT().FindByID(mock.Anything, id).Return(&entity.UserAddress{ID: id, UserID: otherUserID}, nil)

	_, err := uc.UpdateAddress(context.Background(), userID, id, userusecase.AddressInput{})

	assert.Equal(t, "FORBIDDEN", apperr.From(err).Code)
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
