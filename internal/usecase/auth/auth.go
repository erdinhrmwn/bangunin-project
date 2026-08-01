// Package auth implements the account lifecycle: register, email
// verification, login/refresh/logout, password reset (FR-2.1–FR-2.6).
package auth

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/errs"
	"erdinhrmwn/bangunin/internal/domain/repository"
	"erdinhrmwn/bangunin/internal/domain/service"
	"erdinhrmwn/bangunin/pkg/apperr"
	"erdinhrmwn/bangunin/pkg/hash"
	"erdinhrmwn/bangunin/pkg/jwt"
)

const (
	otpTTL         = 10 * time.Minute
	otpMaxFail     = 5
	accessTokenTTL = 15 * time.Minute
	loginFailTTL   = 15 * time.Minute

	otpPurposeVerify = "verify"
	otpPurposeReset  = "reset"
)

type Usecase struct {
	users   repository.UserRepository
	auth    repository.AuthRepository
	email   service.EmailEnqueuer
	jwtSvc  *jwt.Service
	refresh time.Duration
}

func New(users repository.UserRepository, auth repository.AuthRepository, email service.EmailEnqueuer, jwtSvc *jwt.Service, refreshTTL time.Duration) *Usecase {
	return &Usecase{users: users, auth: auth, email: email, jwtSvc: jwtSvc, refresh: refreshTTL}
}

type RegisterInput struct {
	Name     string
	Email    string
	Password string
	Phone    string
	Role     string
}

// Register creates the user and sends a verification OTP (FR-2.1, FR-2.2).
func (u *Usecase) Register(ctx context.Context, in RegisterInput) (*entity.User, error) {
	if in.Role != entity.RoleUser && in.Role != entity.RoleSupplier {
		return nil, apperr.New("VALIDATION_ERROR", "role must be user or supplier", 422)
	}

	if _, err := u.users.FindByEmail(ctx, in.Email); err == nil {
		return nil, apperr.New("EMAIL_TAKEN", "Email is already registered", 409)
	} else if err != errs.ErrNotFound {
		return nil, err
	}

	hashed, err := hash.Hash(in.Password)
	if err != nil {
		return nil, fmt.Errorf("auth: hash password: %w", err)
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("auth: generate id: %w", err)
	}

	roleID := entity.RoleUserID
	if in.Role == entity.RoleSupplier {
		roleID = entity.RoleSupplierID
	}

	usr := &entity.User{
		ID:           id,
		RoleID:       roleID,
		Name:         in.Name,
		Email:        in.Email,
		PasswordHash: hashed,
		Phone:        in.Phone,
		Status:       entity.UserStatusActive,
	}
	if err := u.users.Create(ctx, usr); err != nil {
		return nil, err
	}

	if err := u.sendOTP(ctx, otpPurposeVerify, usr, "Verify your email"); err != nil {
		return nil, err
	}

	return usr, nil
}

// VerifyEmail checks the OTP and sets email_verified_at (FR-2.2).
func (u *Usecase) VerifyEmail(ctx context.Context, email, otp string) error {
	usr, err := u.users.FindByEmail(ctx, email)
	if err != nil {
		return err
	}

	if err := u.checkOTP(ctx, otpPurposeVerify, usr.ID.String(), otp); err != nil {
		return err
	}

	now := time.Now()
	usr.EmailVerifiedAt = &now
	if err := u.users.Update(ctx, usr); err != nil {
		return err
	}
	return u.auth.DeleteOTP(ctx, otpPurposeVerify, usr.ID.String())
}

// ResendOTP re-sends the verification OTP, rate-limited 1x/min per email.
func (u *Usecase) ResendOTP(ctx context.Context, email string) error {
	usr, err := u.users.FindByEmail(ctx, email)
	if err != nil {
		return err
	}

	n, err := u.auth.IncrResend(ctx, email)
	if err != nil {
		return err
	}
	if n > 1 {
		return apperr.New("RATE_LIMITED", "Please wait before requesting another OTP", 429)
	}

	return u.sendOTP(ctx, otpPurposeVerify, usr, "Verify your email")
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

// Login validates credentials under a brute-force guard and issues tokens (FR-2.3).
func (u *Usecase) Login(ctx context.Context, email, password, ip string) (*TokenPair, error) {
	blocked, err := u.auth.IsLoginBlocked(ctx, email, ip)
	if err != nil {
		return nil, err
	}
	if blocked {
		return nil, apperr.New("TOO_MANY_ATTEMPTS", "Too many failed attempts, try again later", 403)
	}

	usr, err := u.users.FindByEmail(ctx, email)
	if err != nil {
		if err == errs.ErrNotFound {
			_, _ = u.auth.IncrLoginFail(ctx, email, ip, loginFailTTL)
			return nil, apperr.New("INVALID_CREDENTIALS", "Invalid email or password", 401)
		}
		return nil, err
	}

	if !hash.Compare(usr.PasswordHash, password) {
		_, _ = u.auth.IncrLoginFail(ctx, email, ip, loginFailTTL)
		return nil, apperr.New("INVALID_CREDENTIALS", "Invalid email or password", 401)
	}

	if usr.EmailVerifiedAt == nil {
		return nil, apperr.New("EMAIL_NOT_VERIFIED", "Email is not verified", 403)
	}
	if usr.Status != entity.UserStatusActive {
		return nil, apperr.New("ACCOUNT_BLOCKED", "Account is suspended or banned", 403)
	}

	return u.issueTokens(ctx, usr)
}

// Refresh rotates the refresh token and issues a new pair (FR-2.4).
func (u *Usecase) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	userID, err := u.auth.GetRefreshToken(ctx, refreshToken)
	if err != nil {
		if err == errs.ErrNotFound {
			return nil, apperr.New("UNAUTHORIZED", "Invalid refresh token", 401)
		}
		return nil, err
	}
	if err := u.auth.DeleteRefreshToken(ctx, refreshToken); err != nil {
		return nil, err
	}

	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, apperr.New("UNAUTHORIZED", "Invalid refresh token", 401)
	}
	usr, err := u.users.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return u.issueTokens(ctx, usr)
}

// Logout deletes the refresh token and blacklists the access token's jti (FR-2.5).
func (u *Usecase) Logout(ctx context.Context, refreshToken, jti string, accessTTLRemaining time.Duration) error {
	if refreshToken != "" {
		if err := u.auth.DeleteRefreshToken(ctx, refreshToken); err != nil {
			return err
		}
	}
	return u.auth.BlacklistJTI(ctx, jti, accessTTLRemaining)
}

// ForgotPassword always returns nil so email existence isn't leaked (FR-2.6).
func (u *Usecase) ForgotPassword(ctx context.Context, email string) error {
	usr, err := u.users.FindByEmail(ctx, email)
	if err != nil {
		return nil //nolint:nilerr // intentional: don't leak whether the email exists
	}
	return u.sendOTP(ctx, otpPurposeReset, usr, "Reset your password")
}

// ResetPassword verifies the OTP, updates the password, and revokes all
// existing sessions (FR-2.6).
func (u *Usecase) ResetPassword(ctx context.Context, email, otp, newPassword string) error {
	usr, err := u.users.FindByEmail(ctx, email)
	if err != nil {
		return err
	}

	if err := u.checkOTP(ctx, otpPurposeReset, usr.ID.String(), otp); err != nil {
		return err
	}

	hashed, err := hash.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("auth: hash password: %w", err)
	}
	usr.PasswordHash = hashed
	if err := u.users.Update(ctx, usr); err != nil {
		return err
	}
	if err := u.auth.RevokeAllRefreshTokens(ctx, usr.ID.String()); err != nil {
		return err
	}
	return u.auth.DeleteOTP(ctx, otpPurposeReset, usr.ID.String())
}

func (u *Usecase) issueTokens(ctx context.Context, usr *entity.User) (*TokenPair, error) {
	access, _, err := u.jwtSvc.GenerateAccess(usr.ID.String(), entity.RoleName(usr.RoleID))
	if err != nil {
		return nil, err
	}

	refreshToken, err := randomToken()
	if err != nil {
		return nil, err
	}
	if err := u.auth.PutRefreshToken(ctx, refreshToken, usr.ID.String(), u.refresh); err != nil {
		return nil, err
	}

	return &TokenPair{AccessToken: access, RefreshToken: refreshToken}, nil
}

func (u *Usecase) sendOTP(ctx context.Context, purpose string, usr *entity.User, subject string) error {
	otp, err := randomOTP()
	if err != nil {
		return err
	}
	if err := u.auth.PutOTP(ctx, purpose, usr.ID.String(), otp, otpTTL); err != nil {
		return err
	}
	return u.email.EnqueueEmail(usr.Email, subject, fmt.Sprintf("Your OTP is %s", otp))
}

func (u *Usecase) checkOTP(ctx context.Context, purpose, userID, otp string) error {
	stored, err := u.auth.GetOTP(ctx, purpose, userID)
	if err != nil {
		if err == errs.ErrNotFound {
			return apperr.New("INVALID_OTP", "OTP is invalid or expired", 422)
		}
		return err
	}

	if stored != otp {
		n, ferr := u.auth.IncrOTPFail(ctx, purpose, userID, otpTTL)
		if ferr != nil {
			return ferr
		}
		if n >= otpMaxFail {
			_ = u.auth.DeleteOTP(ctx, purpose, userID)
		}
		return apperr.New("INVALID_OTP", "OTP is invalid or expired", 422)
	}
	return nil
}

func randomOTP() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	n := (uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])) % 1000000
	return fmt.Sprintf("%06d", n), nil
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}
