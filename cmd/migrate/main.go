// Command migrate runs golang-migrate up/down/create against the configured
// database, and optionally seeds fixed data via -seed.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	pgxmigrate "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	_ "github.com/jackc/pgx/v5/stdlib"

	"erdinhrmwn/bangunin/config"
	"erdinhrmwn/bangunin/internal/infra/database"
	"erdinhrmwn/bangunin/migrations"
	"erdinhrmwn/bangunin/seeds"
)

func main() {
	cmd := flag.String("cmd", "", "up|down|create")
	name := flag.String("name", "", "migration name (for create)")
	seed := flag.Bool("seed", false, "run seeds after migrating")
	demo := flag.Bool("demo", false, "also run demo seed (3 suppliers, 30 products, 1 order per status)")
	flag.Parse()

	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		fatal(err)
	}

	if *cmd == "create" {
		if err := createMigration(*name); err != nil {
			fatal(err)
		}
		return
	}

	if *cmd != "up" && *cmd != "down" {
		fatal(fmt.Errorf("migrate: -cmd must be one of up|down|create"))
	}

	db, err := sql.Open("pgx", cfg.DB.DSN)
	if err != nil {
		fatal(err)
	}
	defer func() { _ = db.Close() }()

	driver, err := pgxmigrate.WithInstance(db, &pgxmigrate.Config{})
	if err != nil {
		fatal(err)
	}

	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		fatal(err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "pgx", driver)
	if err != nil {
		fatal(err)
	}

	switch *cmd {
	case "up":
		err = m.Up()
	case "down":
		err = m.Down()
	}
	if err != nil && err != migrate.ErrNoChange {
		fatal(err)
	}

	if *seed {
		runSeeds(cfg, *demo)
	}
}

func runSeeds(cfg *config.Config, demo bool) {
	ctx := context.Background()
	pool, err := database.NewPool(ctx, cfg.DB)
	if err != nil {
		fatal(err)
	}
	defer pool.Close()

	if err := seeds.Roles(ctx, pool); err != nil {
		fatal(err)
	}
	if err := seeds.Admin(ctx, pool); err != nil {
		fatal(err)
	}
	if demo {
		if err := seeds.Demo(ctx, pool); err != nil {
			fatal(err)
		}
	}
}

func createMigration(name string) error {
	if name == "" {
		return fmt.Errorf("migrate: -name is required for create")
	}

	entries, err := os.ReadDir("migrations")
	if err != nil {
		return err
	}
	next := 1
	for _, e := range entries {
		parts := strings.SplitN(e.Name(), "_", 2)
		if n, err := strconv.Atoi(parts[0]); err == nil && n >= next {
			next = n + 1
		}
	}
	base := fmt.Sprintf("migrations/%06d_%s", next, name)
	for _, suffix := range []string{"up", "down"} {
		path := filepath.Clean(base + "." + suffix + ".sql")
		if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
			return err
		}
		fmt.Println("created", path)
	}
	return nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
