package repository

import (
	"context"
	"time"
)

// AuthRepository holds the Redis-backed auth state that isn't durable
// (OTPs, refresh tokens, blacklist, rate-limit counters) per FR-2.2–2.6.
type AuthRepository interface {
	// OTP: purpose is "verify" (FR-2.2) or "reset" (FR-2.6).
	PutOTP(ctx context.Context, purpose, userID, otp string, ttl time.Duration) error
	GetOTP(ctx context.Context, purpose, userID string) (string, error)
	DeleteOTP(ctx context.Context, purpose, userID string) error
	IncrOTPFail(ctx context.Context, purpose, userID string, ttl time.Duration) (int64, error)

	// IncrResend enforces the 1x/min resend-OTP limit per email.
	IncrResend(ctx context.Context, email string) (int64, error)

	// Refresh tokens: opaque string -> userID.
	PutRefreshToken(ctx context.Context, token, userID string, ttl time.Duration) error
	GetRefreshToken(ctx context.Context, token string) (string, error)
	DeleteRefreshToken(ctx context.Context, token string) error
	// RevokeAllRefreshTokens invalidates every refresh token issued to userID,
	// e.g. on password reset (FR-2.6).
	RevokeAllRefreshTokens(ctx context.Context, userID string) error

	// BlacklistJTI marks an access token's jti revoked until it would have expired.
	BlacklistJTI(ctx context.Context, jti string, ttl time.Duration) error
	IsJTIBlacklisted(ctx context.Context, jti string) (bool, error)

	// Brute force guard: 10 failures/15min per email+IP (FR-2.3).
	IncrLoginFail(ctx context.Context, email, ip string, ttl time.Duration) (int64, error)
	IsLoginBlocked(ctx context.Context, email, ip string) (bool, error)
}
