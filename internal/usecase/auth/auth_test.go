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
	"erdinhrmwn/bangunin/pkg/apperr"
	"erdinhrmwn/bangunin/pkg/hash"
	"erdinhrmwn/bangunin/pkg/jwt"

	svcmocks "erdinhrmwn/bangunin/internal/domain/service/mocks"
	authusecase "erdinhrmwn/bangunin/internal/usecase/auth"
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

// ResetPassword: success revokes all existing refresh token sessions.
func TestResetPassword_RevokesSessions(t *testing.T) {
	uc, users, authRepo, _ := newUsecase(t)
	usr := &entity.User{ID: mustUUID(t), Email: "a@example.com"}
	users.EXPECT().FindByEmail(mock.Anything, "a@example.com").Return(usr, nil)
	authRepo.EXPECT().GetOTP(mock.Anything, "reset", usr.ID.String()).Return("123456", nil)
	users.EXPECT().Update(mock.Anything, usr).Return(nil)
	authRepo.EXPECT().RevokeAllRefreshTokens(mock.Anything, usr.ID.String()).Return(nil)
	authRepo.EXPECT().DeleteOTP(mock.Anything, "reset", usr.ID.String()).Return(nil)

	err := uc.ResetPassword(context.Background(), "a@example.com", "123456", "new-password1")

	assert.NoError(t, err)
}

// Register: success path creates the user and sends a verification OTP.
func TestRegister_Success(t *testing.T) {
	uc, users, authRepo, email := newUsecase(t)
	users.EXPECT().FindByEmail(mock.Anything, "new@example.com").Return(nil, errs.ErrNotFound)
	users.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)
	authRepo.EXPECT().PutOTP(mock.Anything, "verify", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	email.EXPECT().EnqueueEmail(mock.Anything, "Verify your email", mock.Anything).Return(nil)

	usr, err := uc.Register(context.Background(), authusecase.RegisterInput{
		Name: "A", Email: "new@example.com", Password: "password1", Role: entity.RoleUser,
	})

	assert.NoError(t, err)
	assert.Equal(t, "new@example.com", usr.Email)
}

// ResendOTP: second call within the window is rate limited.
func TestResendOTP_RateLimited(t *testing.T) {
	uc, users, authRepo, _ := newUsecase(t)
	usr := &entity.User{ID: mustUUID(t), Email: "a@example.com"}
	users.EXPECT().FindByEmail(mock.Anything, "a@example.com").Return(usr, nil)
	authRepo.EXPECT().IncrResend(mock.Anything, "a@example.com").Return(int64(2), nil)

	err := uc.ResendOTP(context.Background(), "a@example.com")

	assert.Equal(t, "RATE_LIMITED", appErrCode(t, err))
}

// ResendOTP: first call re-sends the OTP.
func TestResendOTP_Success(t *testing.T) {
	uc, users, authRepo, email := newUsecase(t)
	usr := &entity.User{ID: mustUUID(t), Email: "a@example.com"}
	users.EXPECT().FindByEmail(mock.Anything, "a@example.com").Return(usr, nil)
	authRepo.EXPECT().IncrResend(mock.Anything, "a@example.com").Return(int64(1), nil)
	authRepo.EXPECT().PutOTP(mock.Anything, "verify", usr.ID.String(), mock.Anything, mock.Anything).Return(nil)
	email.EXPECT().EnqueueEmail(usr.Email, "Verify your email", mock.Anything).Return(nil)

	err := uc.ResendOTP(context.Background(), "a@example.com")

	assert.NoError(t, err)
}

// Login: success issues an access/refresh token pair.
func TestLogin_Success(t *testing.T) {
	uc, users, authRepo, _ := newUsecase(t)
	hashed, err := hash.Hash("correct-pass1")
	assert.NoError(t, err)
	now := time.Now()
	usr := &entity.User{ID: mustUUID(t), Email: "a@example.com", PasswordHash: hashed, EmailVerifiedAt: &now, Status: entity.UserStatusActive, RoleID: entity.RoleUserID}

	authRepo.EXPECT().IsLoginBlocked(mock.Anything, "a@example.com", "1.2.3.4").Return(false, nil)
	users.EXPECT().FindByEmail(mock.Anything, "a@example.com").Return(usr, nil)
	authRepo.EXPECT().PutRefreshToken(mock.Anything, mock.Anything, usr.ID.String(), mock.Anything).Return(nil)

	pair, err := uc.Login(context.Background(), "a@example.com", "correct-pass1", "1.2.3.4")

	assert.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
}

// Refresh: invalid/expired token -> 401.
func TestRefresh_InvalidToken(t *testing.T) {
	uc, _, authRepo, _ := newUsecase(t)
	authRepo.EXPECT().GetRefreshToken(mock.Anything, "bad-token").Return("", errs.ErrNotFound)

	_, err := uc.Refresh(context.Background(), "bad-token")

	assert.Equal(t, "UNAUTHORIZED", appErrCode(t, err))
}

// Refresh: success rotates the token and issues a new pair.
func TestRefresh_Success(t *testing.T) {
	uc, users, authRepo, _ := newUsecase(t)
	usr := &entity.User{ID: mustUUID(t), Email: "a@example.com", RoleID: entity.RoleUserID}
	authRepo.EXPECT().GetRefreshToken(mock.Anything, "old-token").Return(usr.ID.String(), nil)
	authRepo.EXPECT().DeleteRefreshToken(mock.Anything, "old-token").Return(nil)
	users.EXPECT().FindByID(mock.Anything, usr.ID).Return(usr, nil)
	authRepo.EXPECT().PutRefreshToken(mock.Anything, mock.Anything, usr.ID.String(), mock.Anything).Return(nil)

	pair, err := uc.Refresh(context.Background(), "old-token")

	assert.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
}

// Logout: deletes the refresh token and blacklists the access token jti.
func TestLogout_Success(t *testing.T) {
	uc, _, authRepo, _ := newUsecase(t)
	authRepo.EXPECT().DeleteRefreshToken(mock.Anything, "some-token").Return(nil)
	authRepo.EXPECT().BlacklistJTI(mock.Anything, "jti-1", 5*time.Minute).Return(nil)

	err := uc.Logout(context.Background(), "some-token", "jti-1", 5*time.Minute)

	assert.NoError(t, err)
}

// Logout: empty refresh token still blacklists the access token jti.
func TestLogout_EmptyRefreshToken(t *testing.T) {
	uc, _, authRepo, _ := newUsecase(t)
	authRepo.EXPECT().BlacklistJTI(mock.Anything, "jti-1", 5*time.Minute).Return(nil)

	err := uc.Logout(context.Background(), "", "jti-1", 5*time.Minute)

	assert.NoError(t, err)
}

// ForgotPassword: unknown email doesn't leak existence, returns nil.
func TestForgotPassword_UnknownEmail(t *testing.T) {
	uc, users, _, _ := newUsecase(t)
	users.EXPECT().FindByEmail(mock.Anything, "ghost@example.com").Return(nil, errs.ErrNotFound)

	err := uc.ForgotPassword(context.Background(), "ghost@example.com")

	assert.NoError(t, err)
}

// ForgotPassword: known email sends a reset OTP.
func TestForgotPassword_Success(t *testing.T) {
	uc, users, authRepo, email := newUsecase(t)
	usr := &entity.User{ID: mustUUID(t), Email: "a@example.com"}
	users.EXPECT().FindByEmail(mock.Anything, "a@example.com").Return(usr, nil)
	authRepo.EXPECT().PutOTP(mock.Anything, "reset", usr.ID.String(), mock.Anything, mock.Anything).Return(nil)
	email.EXPECT().EnqueueEmail(usr.Email, "Reset your password", mock.Anything).Return(nil)

	err := uc.ForgotPassword(context.Background(), "a@example.com")

	assert.NoError(t, err)
}

func mustUUID(t *testing.T) uuid.UUID {
	t.Helper()
	u, err := uuid.NewV7()
	assert.NoError(t, err)
	return u
}
