package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/errs"
	"erdinhrmwn/bangunin/internal/domain/repository/mocks"
	svcmocks "erdinhrmwn/bangunin/internal/domain/service/mocks"
	authusecase "erdinhrmwn/bangunin/internal/usecase/auth"
	"erdinhrmwn/bangunin/pkg/apperr"
	"erdinhrmwn/bangunin/pkg/hash"
	"erdinhrmwn/bangunin/pkg/jwt"
)

func newUsecase(t *testing.T) (*authusecase.Usecase, *mocks.MockUserRepository, *mocks.MockAuthRepository, *svcmocks.MockEmailEnqueuer) {
	t.Helper()
	users := mocks.NewMockUserRepository(t)
	authRepo := mocks.NewMockAuthRepository(t)
	email := svcmocks.NewMockEmailEnqueuer(t)
	jwtSvc := jwt.NewService("test-secret", 15*time.Minute)
	uc := authusecase.New(users, authRepo, email, jwtSvc, 7*24*time.Hour)
	return uc, users, authRepo, email
}

func appErrCode(t *testing.T, err error) string {
	t.Helper()
	return apperr.From(err).Code
}

// Register: duplicate email -> 409 EMAIL_TAKEN.
func TestRegister_DuplicateEmail(t *testing.T) {
	uc, users, _, _ := newUsecase(t)
	existing := &entity.User{ID: mustUUID(t), Email: "taken@example.com"}
	users.EXPECT().FindByEmail(mock.Anything, "taken@example.com").Return(existing, nil)

	_, err := uc.Register(context.Background(), authusecase.RegisterInput{
		Name: "A", Email: "taken@example.com", Password: "password1", Role: entity.RoleUser,
	})

	assert.Equal(t, "EMAIL_TAKEN", appErrCode(t, err))
}

// Register: admin role is rejected.
func TestRegister_AdminRoleRejected(t *testing.T) {
	uc, _, _, _ := newUsecase(t)

	_, err := uc.Register(context.Background(), authusecase.RegisterInput{
		Name: "A", Email: "a@example.com", Password: "password1", Role: entity.RoleAdmin,
	})

	assert.Equal(t, "VALIDATION_ERROR", appErrCode(t, err))
}

// VerifyEmail: wrong OTP -> INVALID_OTP, and increments the fail counter.
func TestVerifyEmail_WrongOTP(t *testing.T) {
	uc, users, authRepo, _ := newUsecase(t)
	usr := &entity.User{ID: mustUUID(t), Email: "a@example.com"}
	users.EXPECT().FindByEmail(mock.Anything, "a@example.com").Return(usr, nil)
	authRepo.EXPECT().GetOTP(mock.Anything, "verify", usr.ID.String()).Return("123456", nil)
	authRepo.EXPECT().IncrOTPFail(mock.Anything, "verify", usr.ID.String(), mock.Anything).Return(int64(1), nil)

	err := uc.VerifyEmail(context.Background(), "a@example.com", "000000")

	assert.Equal(t, "INVALID_OTP", appErrCode(t, err))
}

// VerifyEmail: expired/missing OTP -> INVALID_OTP.
func TestVerifyEmail_ExpiredOTP(t *testing.T) {
	uc, users, authRepo, _ := newUsecase(t)
	usr := &entity.User{ID: mustUUID(t), Email: "a@example.com"}
	users.EXPECT().FindByEmail(mock.Anything, "a@example.com").Return(usr, nil)
	authRepo.EXPECT().GetOTP(mock.Anything, "verify", usr.ID.String()).Return("", errs.ErrNotFound)

	err := uc.VerifyEmail(context.Background(), "a@example.com", "000000")

	assert.Equal(t, "INVALID_OTP", appErrCode(t, err))
}

// VerifyEmail: 5th consecutive wrong OTP deletes it (further attempts see it as expired).
func TestVerifyEmail_FifthFailDeletesOTP(t *testing.T) {
	uc, users, authRepo, _ := newUsecase(t)
	usr := &entity.User{ID: mustUUID(t), Email: "a@example.com"}
	users.EXPECT().FindByEmail(mock.Anything, "a@example.com").Return(usr, nil)
	authRepo.EXPECT().GetOTP(mock.Anything, "verify", usr.ID.String()).Return("123456", nil)
	authRepo.EXPECT().IncrOTPFail(mock.Anything, "verify", usr.ID.String(), mock.Anything).Return(int64(5), nil)
	authRepo.EXPECT().DeleteOTP(mock.Anything, "verify", usr.ID.String()).Return(nil)

	err := uc.VerifyEmail(context.Background(), "a@example.com", "000000")

	assert.Equal(t, "INVALID_OTP", appErrCode(t, err))
}

// Login: brute force lock after threshold reached.
func TestLogin_BruteForceLocked(t *testing.T) {
	uc, _, authRepo, _ := newUsecase(t)
	authRepo.EXPECT().IsLoginBlocked(mock.Anything, "a@example.com", "1.2.3.4").Return(true, nil)

	_, err := uc.Login(context.Background(), "a@example.com", "password1", "1.2.3.4")

	assert.Equal(t, "TOO_MANY_ATTEMPTS", appErrCode(t, err))
}

// Login: wrong password increments the brute-force counter and fails.
func TestLogin_WrongPasswordIncrementsFailCounter(t *testing.T) {
	uc, users, authRepo, _ := newUsecase(t)
	hashed, err := hash.Hash("correct-pass1")
	assert.NoError(t, err)
	usr := &entity.User{ID: mustUUID(t), Email: "a@example.com", PasswordHash: hashed}

	authRepo.EXPECT().IsLoginBlocked(mock.Anything, "a@example.com", "1.2.3.4").Return(false, nil)
	users.EXPECT().FindByEmail(mock.Anything, "a@example.com").Return(usr, nil)
	authRepo.EXPECT().IncrLoginFail(mock.Anything, "a@example.com", "1.2.3.4", mock.Anything).Return(int64(1), nil)

	_, err = uc.Login(context.Background(), "a@example.com", "wrong-pass", "1.2.3.4")

	assert.Equal(t, "INVALID_CREDENTIALS", appErrCode(t, err))
}

func mustUUID(t *testing.T) uuid.UUID {
	t.Helper()
	u, err := uuid.NewV7()
	assert.NoError(t, err)
	return u
}
