// Package seeds holds idempotent seed data run by cmd/migrate -seed.
package seeds

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"erdinhrmwn/bangunin/internal/domain/entity"
)

// Roles seeds the fixed role set. ON CONFLICT DO NOTHING makes re-running
// safe (AC-1.c requires seed to run twice without error/duplicates).
func Roles(ctx context.Context, db *pgxpool.Pool) error {
	roles := []entity.Role{
		{ID: 1, Name: entity.RoleAdmin},
		{ID: 2, Name: entity.RoleSupplier},
		{ID: 3, Name: entity.RoleUser},
	}

	for _, r := range roles {
		_, err := db.Exec(ctx,
			`INSERT INTO roles (id, name) VALUES ($1, $2) ON CONFLICT (id) DO NOTHING`,
			r.ID, r.Name,
		)
		if err != nil {
			return fmt.Errorf("seeds: roles: %w", err)
		}
	}
	return nil
}
