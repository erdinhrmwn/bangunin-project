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

func TestMe_Success(t *testing.T) {
	uc, users := newUsecase(t)
	id := mustUUID(t)
	usr := &entity.User{ID: id, Name: "Budi"}
	users.EXPECT().FindByID(mock.Anything, id).Return(usr, nil)

	got, err := uc.Me(context.Background(), id)

	assert.NoError(t, err)
	assert.Equal(t, usr, got)
}

func TestMe_NotFound(t *testing.T) {
	uc, users := newUsecase(t)
	id := mustUUID(t)
	users.EXPECT().FindByID(mock.Anything, id).Return(nil, assert.AnError)

	_, err := uc.Me(context.Background(), id)

	assert.Error(t, err)
}

func TestUpdateProfile_Success(t *testing.T) {
	uc, users := newUsecase(t)
	id := mustUUID(t)
	usr := &entity.User{ID: id, Name: "Old", Phone: "0800"}
	users.EXPECT().FindByID(mock.Anything, id).Return(usr, nil)
	users.EXPECT().Update(mock.Anything, usr).Return(nil)

	got, err := uc.UpdateProfile(context.Background(), id, "New", "0811")

	assert.NoError(t, err)
	assert.Equal(t, "New", got.Name)
	assert.Equal(t, "0811", got.Phone)
}

func TestUpdateProfile_NotFound(t *testing.T) {
	uc, users := newUsecase(t)
	id := mustUUID(t)
	users.EXPECT().FindByID(mock.Anything, id).Return(nil, assert.AnError)

	_, err := uc.UpdateProfile(context.Background(), id, "New", "0811")

	assert.Error(t, err)
}

func TestUpdateNotificationSettings_Success(t *testing.T) {
	uc, users := newUsecase(t)
	id := mustUUID(t)
	usr := &entity.User{ID: id, EmailMarketing: false}
	users.EXPECT().FindByID(mock.Anything, id).Return(usr, nil)
	users.EXPECT().Update(mock.Anything, usr).Return(nil)

	got, err := uc.UpdateNotificationSettings(context.Background(), id, true)

	assert.NoError(t, err)
	assert.True(t, got.EmailMarketing)
}

func TestUpdateNotificationSettings_NotFound(t *testing.T) {
	uc, users := newUsecase(t)
	id := mustUUID(t)
	users.EXPECT().FindByID(mock.Anything, id).Return(nil, assert.AnError)

	_, err := uc.UpdateNotificationSettings(context.Background(), id, true)

	assert.Error(t, err)
}

func TestChangePassword_UserNotFound(t *testing.T) {
	uc, users := newUsecase(t)
	id := mustUUID(t)
	users.EXPECT().FindByID(mock.Anything, id).Return(nil, assert.AnError)

	err := uc.ChangePassword(context.Background(), id, "old", "new")

	assert.Error(t, err)
}

func TestListAddresses_Success(t *testing.T) {
	uc, _, addresses := newUsecaseWithAddresses(t)
	userID := mustUUID(t)
	want := []*entity.UserAddress{{ID: mustUUID(t), UserID: userID}}
	addresses.EXPECT().ListByUserID(mock.Anything, userID).Return(want, nil)

	got, err := uc.ListAddresses(context.Background(), userID)

	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

// UpdateAddress: owned address updates fields and, when default, clears
// other defaults.
func TestUpdateAddress_Default(t *testing.T) {
	uc, _, addresses := newUsecaseWithAddresses(t)
	userID, id := mustUUID(t), mustUUID(t)
	existing := &entity.UserAddress{ID: id, UserID: userID, Label: "Old"}
	addresses.EXPECT().FindByID(mock.Anything, id).Return(existing, nil)
	addresses.EXPECT().Update(mock.Anything, existing).Return(nil)
	addresses.EXPECT().ClearDefault(mock.Anything, userID, id).Return(nil)

	got, err := uc.UpdateAddress(context.Background(), userID, id, userusecase.AddressInput{Label: "New", IsDefault: true})

	assert.NoError(t, err)
	assert.Equal(t, "New", got.Label)
}

// UpdateAddress: address not found propagates the repository error.
func TestUpdateAddress_NotFound(t *testing.T) {
	uc, _, addresses := newUsecaseWithAddresses(t)
	userID, id := mustUUID(t), mustUUID(t)
	addresses.EXPECT().FindByID(mock.Anything, id).Return(nil, assert.AnError)

	_, err := uc.UpdateAddress(context.Background(), userID, id, userusecase.AddressInput{})

	assert.Error(t, err)
}

func TestDeleteAddress_Success(t *testing.T) {
	uc, _, addresses := newUsecaseWithAddresses(t)
	userID, id := mustUUID(t), mustUUID(t)
	addresses.EXPECT().FindByID(mock.Anything, id).Return(&entity.UserAddress{ID: id, UserID: userID}, nil)
	addresses.EXPECT().Delete(mock.Anything, id).Return(nil)

	err := uc.DeleteAddress(context.Background(), userID, id)

	assert.NoError(t, err)
}

func TestDeleteAddress_NotOwner(t *testing.T) {
	uc, _, addresses := newUsecaseWithAddresses(t)
	userID, otherUserID, id := mustUUID(t), mustUUID(t), mustUUID(t)
	addresses.EXPECT().FindByID(mock.Anything, id).Return(&entity.UserAddress{ID: id, UserID: otherUserID}, nil)

	err := uc.DeleteAddress(context.Background(), userID, id)

	assert.Equal(t, "FORBIDDEN", apperr.From(err).Code)
}

// CreateAddress: repository error on Create propagates without ClearDefault.
func TestCreateAddress_CreateError(t *testing.T) {
	uc, _, addresses := newUsecaseWithAddresses(t)
	userID := mustUUID(t)
	addresses.EXPECT().Create(mock.Anything, mock.AnythingOfType("*entity.UserAddress")).Return(assert.AnError)

	_, err := uc.CreateAddress(context.Background(), userID, userusecase.AddressInput{IsDefault: true})

	assert.Error(t, err)
}
