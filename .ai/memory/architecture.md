# Hellfire Gatekeeper — Architecture & Philosophy

This document is the source of truth for local AI architecture review.
Judge code against these rules. Prefer project conventions over generic Go advice.

## Purpose

Backend API for a multi-tenant bakery management system (products, orders, auth, images, subscriptions).

## Layering (Clean Architecture)

Allowed dependency flow only:

```
handlers → services → repository
```

- `internal/handlers/` — HTTP adapters: parse request, validate, call services, map errors to HTTP.
- `internal/services/` — business logic and orchestration.
- `internal/repository/` — data access (SQL). No business rules that belong in services.
- `internal/middleware/` — auth, tenant, subscription, rate limit.
- `internal/handlers/validators/` — request/input validation.
- `model/` — domain/data models (no HTTP, no SQL drivers).
- `cmd/api/` — wiring / composition root only.
- `migrations/` — schema changes via golang-migrate.
- `tests/` — integration tests (testcontainers + real PostgreSQL).

### Forbidden

- Handlers calling repositories directly (bypass services).
- Services importing handlers or middleware.
- Repositories importing services or handlers.
- Cross-layer shortcuts or circular dependencies.
- Creating concrete external clients inside handlers/services (use interfaces + constructor injection).

## Dependency injection

- Inject dependencies via constructors.
- External deps (DB, email, storage, clocks) behind interfaces.
- Never `NewX()` hidden inside business methods when it should be injected.

## Errors

- Prefer domain/sentinel errors from `internal/errors`.
- Wrap with context: `fmt.Errorf("failed to create order: %w", err)`.
- Handlers must map known errors to the correct HTTP status (e.g. HTTPError / BadRequest helpers).
- Do not leak internal/DB details in HTTP responses.

## Logging

- Use structured logger (`zerolog` via `internal/logger`), not `fmt.Printf` / `log.Println` for app logs.
- Include relevant context: `user_id`, `tenant_id`, `order_id`, operation name.
- Levels: debug, info, warn, error.
- Never log secrets, passwords, tokens, or raw credentials.

## Security

- Validate all user input (validators / handler validation).
- Protected routes must use auth middleware (JWT).
- Passwords: bcrypt only; never store plaintext.
- No hardcoded secrets; use env vars.
- Respect tenant isolation — do not cross tenant boundaries in queries or auth checks.

## Database

- Schema changes only through migrations.
- Use transactions for multi-step writes.
- Keep history tables where the domain already does (`products_history`, `orders_history`, etc.).
- DB naming: `snake_case`; descriptive prefixes (`id_`, `created_on`, `modified_on`).
- Avoid N+1 queries; prefer explicit joins/batching when listing related data.

## Testing

- TDD mindset for new behavior: tests should cover the change.
- Unit tests for repositories with `sqlmock`.
- Integration tests with testcontainers for full flows.
- Naming: `Test[Struct]_[Method]_[Scenario]`.
- Use `testify/assert` or `testify/require`.
- Cover happy path, edge cases, and error paths for new logic.

## Style

- Exported: PascalCase. Unexported: camelCase.
- Code, comments, and public docs in English.
- Files: `[domain].go`, `[domain]_test.go`.
- Keep functions small and single-purpose.

## Pagination / lists

- List endpoints follow existing cursor/keyset pagination patterns (`limit`, `cursor`, `next_cursor`).
- Do not invent offset pagination for new list APIs unless explicitly required and documented.
