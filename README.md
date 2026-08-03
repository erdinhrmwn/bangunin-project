# Bangunin

**Vertical marketplace for building materials & construction supplies in Indonesia.**

Suppliers (distributors/hardware stores) sell to contractors, builders, and project owners. Built around the things that make this vertical different from a generic marketplace: high-value orders, weight-based shipping for heavy materials, escrow payments, and orders split per supplier.

![Go](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go&logoColor=white)
![Fiber](https://img.shields.io/badge/Fiber-v3-000000?logo=fiber&logoColor=white)
![Status](https://img.shields.io/badge/status-planning-yellow)
![License](https://img.shields.io/badge/license-TBD-lightgrey)

> **Status: planning stage.** Architecture and requirements are fully specified; implementation has not started yet. See [PROGRESS.md](PROGRESS.md) for the phase-by-phase checklist.

## Why it's not just another marketplace

- **Split checkout** — one cart, multiple suppliers → one invoice, N orders, shipping cost calculated per supplier.
- **Weight-aware shipping** — chargeable weight = `max(actual weight, volumetric)`, with a courier or supplier-fleet option per order.
- **Escrow + auditable ledger** — funds settle to the platform first; payouts run through an append-only ledger, no `UPDATE`/`DELETE` ever.
- **Stock reservation, not instant deduction** — checkout locks stock via Redis, converts to a real deduction only once paid.

## Tech Stack

| Layer | Choice |
|---|---|
| Language / HTTP | Go 1.25+ · [Fiber v3](https://gofiber.io) |
| Database | PostgreSQL 16 (pgx + sqlc) |
| Cache / Queue backend | Redis 7 |
| Background jobs | Asynq |
| Payment | Xendit |
| Notification | Mailjet |
| Shipping | RajaOngkir |
| Object storage | Cloudflare R2 (MinIO for dev) |

External providers (Xendit, Mailjet, RajaOngkir) are wired in as internal adapter packages behind `domain/service` interfaces — called in-process, no separate microservices until there's a real reason to split one out.

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

| Phase | Scope |
|---|---|
| 1 | Foundation — skeleton, config, DB/Redis, middleware |
| 2 | Auth + RBAC |
| 3 | Supplier onboarding |
| 4 | Catalog |
| 5 | Cart, checkout & payment (ends at a `paid` order) |
| 6 | Order lifecycle & shipment (paid → shipped → delivered → completed) |
| 7 | Post-transaction (reviews, payout/ledger) |
| 8 | Supplementary features (wishlist, banner, reports) |
| 9 | Production hardening (security, observability, quality, docs) |

## License

Not yet decided.
