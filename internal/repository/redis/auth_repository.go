package redis

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"

	"erdinhrmwn/bangunin/internal/domain/errs"
)

// loginFailLimit mirrors FR-2.3's "10 failures/15min per email+IP" guard.
const loginFailLimit = 10

type AuthRepository struct {
	client *redis.Client
}

func NewAuthRepository(client *redis.Client) *AuthRepository {
	return &AuthRepository{client: client}
}

func (r *AuthRepository) PutOTP(ctx context.Context, purpose, userID, otp string, ttl time.Duration) error {
	return r.client.Set(ctx, otpKey(purpose, userID), otp, ttl).Err()
}

func (r *AuthRepository) GetOTP(ctx context.Context, purpose, userID string) (string, error) {
	v, err := r.client.Get(ctx, otpKey(purpose, userID)).Result()
	if errors.Is(err, redis.Nil) {
		return "", errs.ErrNotFound
	}
	return v, err
}

func (r *AuthRepository) DeleteOTP(ctx context.Context, purpose, userID string) error {
	return r.client.Del(ctx, otpKey(purpose, userID), otpFailKey(purpose, userID)).Err()
}

func (r *AuthRepository) IncrOTPFail(ctx context.Context, purpose, userID string, ttl time.Duration) (int64, error) {
	return incrWithTTL(ctx, r.client, otpFailKey(purpose, userID), ttl)
}

func (r *AuthRepository) IncrResend(ctx context.Context, email string) (int64, error) {
	return incrWithTTL(ctx, r.client, "otp:resend:"+email, time.Minute)
}

func (r *AuthRepository) PutRefreshToken(ctx context.Context, token, userID string, ttl time.Duration) error {
	return r.client.Set(ctx, "refresh:"+token, userID, ttl).Err()
}

func (r *AuthRepository) GetRefreshToken(ctx context.Context, token string) (string, error) {
	v, err := r.client.Get(ctx, "refresh:"+token).Result()
	if errors.Is(err, redis.Nil) {
		return "", errs.ErrNotFound
	}
	return v, err
}

func (r *AuthRepository) DeleteRefreshToken(ctx context.Context, token string) error {
	return r.client.Del(ctx, "refresh:"+token).Err()
}

func (r *AuthRepository) BlacklistJTI(ctx context.Context, jti string, ttl time.Duration) error {
	if ttl <= 0 {
		return nil
	}
	return r.client.Set(ctx, "blacklist:"+jti, "1", ttl).Err()
}

func (r *AuthRepository) IsJTIBlacklisted(ctx context.Context, jti string) (bool, error) {
	n, err := r.client.Exists(ctx, "blacklist:"+jti).Result()
	return n > 0, err
}

func (r *AuthRepository) IncrLoginFail(ctx context.Context, email, ip string, ttl time.Duration) (int64, error) {
	return incrWithTTL(ctx, r.client, "loginfail:"+email+":"+ip, ttl)
}

func (r *AuthRepository) IsLoginBlocked(ctx context.Context, email, ip string) (bool, error) {
	n, err := r.client.Get(ctx, "loginfail:"+email+":"+ip).Int64()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return n >= loginFailLimit, nil
}

func otpKey(purpose, userID string) string     { return "otp:" + purpose + ":" + userID }
func otpFailKey(purpose, userID string) string { return "otp:" + purpose + ":" + userID + ":fail" }

func incrWithTTL(ctx context.Context, client *redis.Client, key string, ttl time.Duration) (int64, error) {
	n, err := client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if n == 1 {
		if err := client.Expire(ctx, key, ttl).Err(); err != nil {
			return 0, err
		}
	}
	return n, nil
}
