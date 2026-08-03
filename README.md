# Bangunin

Vertical marketplace for building materials & construction supplies in Indonesia. Suppliers (distributors/hardware stores) sell to contractors, builders, and project owners. Handles high-value orders, weight-based shipping for heavy materials, escrow payments, and per-supplier order splitting.

> **Status: planning stage.** Architecture and requirements are fully specified; implementation has not started yet (see [PROGRESS.md](PROGRESS.md)).

## Tech Stack

- Go 1.25+ · [Fiber v3](https://gofiber.io) (HTTP)
- PostgreSQL 16 (pgx + sqlc) · Redis 7 (cache, session, rate limit, stock lock)
- Asynq (background jobs & scheduler)
- gRPC to external payment-service (Xendit) & notification-service (Mailjet)
- RajaOngkir for shipping cost & tracking
- Cloudflare R2 / MinIO (object storage)

## Docs

| File | Purpose |
|---|---|
| [CLAUDE.md](CLAUDE.md) | Project context: architecture, tech stack, conventions, git workflow |
| [docs/requirements.md](docs/requirements.md) | Functional requirements & acceptance criteria per phase |
| [docs/erd.md](docs/erd.md) | ER diagrams & process flows |
| [docs/project-structure.md](docs/project-structure.md) | Module list & directory layout |
| [docs/packages-reference.md](docs/packages-reference.md) | API reference/patterns per dependency |
| [PROGRESS.md](PROGRESS.md) | Phase-by-phase implementation checklist |

## Roadmap

1. Foundation — skeleton, config, DB/Redis, middleware
2. Auth + RBAC
3. Supplier onboarding
4. Catalog
5. Core transactions (cart → checkout → payment → order lifecycle)
6. Post-transaction (reviews, payout/ledger)
7. Supplementary & hardening

## License

Not yet decided.
