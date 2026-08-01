package seeds

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/pkg/hash"
)

// Admin seeds one admin user from SEED_ADMIN_EMAIL/SEED_ADMIN_PASSWORD env
// vars. No-op if either is unset. ON CONFLICT DO NOTHING on email keeps
// re-runs safe (AC-1.c).
func Admin(ctx context.Context, db *pgxpool.Pool) error {
	email := os.Getenv("SEED_ADMIN_EMAIL")
	password := os.Getenv("SEED_ADMIN_PASSWORD")
	if email == "" || password == "" {
		return nil
	}

	hashed, err := hash.Hash(password)
	if err != nil {
		return fmt.Errorf("seeds: admin: %w", err)
	}

	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("seeds: admin: %w", err)
	}

	_, err = db.Exec(ctx,
		`INSERT INTO users (id, role_id, name, email, password_hash)
		 VALUES ($1, $2, $3, $4, $5) ON CONFLICT (email) DO NOTHING`,
		id, entity.RoleAdminID, "Admin", email, hashed,
	)
	if err != nil {
		return fmt.Errorf("seeds: admin: %w", err)
	}
	return nil
}
