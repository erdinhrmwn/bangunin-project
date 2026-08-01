package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/errs"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, u *entity.User) error {
	const q = `
		INSERT INTO users (id, role_id, name, email, password_hash, phone, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.Exec(ctx, q, u.ID, u.RoleID, u.Name, u.Email, u.PasswordHash, u.Phone, u.Status)
	if err != nil {
		return fmt.Errorf("postgres: create user: %w", err)
	}
	return nil
}

func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	const q = `
		SELECT id, role_id, name, email, password_hash, phone, status, email_verified_at, email_marketing, created_at
		FROM users WHERE id = $1
	`
	return r.scanOne(ctx, q, id)
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	const q = `
		SELECT id, role_id, name, email, password_hash, phone, status, email_verified_at, email_marketing, created_at
		FROM users WHERE email = $1
	`
	return r.scanOne(ctx, q, email)
}

func (r *UserRepository) Update(ctx context.Context, u *entity.User) error {
	const q = `
		UPDATE users
		SET name = $2, phone = $3, password_hash = $4, status = $5, email_verified_at = $6, email_marketing = $7
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, q, u.ID, u.Name, u.Phone, u.PasswordHash, u.Status, u.EmailVerifiedAt, u.EmailMarketing)
	if err != nil {
		return fmt.Errorf("postgres: update user: %w", err)
	}
	return nil
}

func (r *UserRepository) scanOne(ctx context.Context, query string, arg any) (*entity.User, error) {
	var u entity.User
	err := r.db.QueryRow(ctx, query, arg).Scan(
		&u.ID, &u.RoleID, &u.Name, &u.Email, &u.PasswordHash, &u.Phone, &u.Status,
		&u.EmailVerifiedAt, &u.EmailMarketing, &u.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errs.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: find user: %w", err)
	}
	return &u, nil
}
