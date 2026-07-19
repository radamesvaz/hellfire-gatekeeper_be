# Architecture review checklist

Use this as a fast scan against the diff. Flag only issues supported by the guidelines.

## Layers & DI

- [ ] No handler → repository bypass
- [ ] No upward imports (repo/service → handler)
- [ ] External deps behind interfaces + constructor injection
- [ ] No `NewConcrete()` inside business logic that should be injected

## Handlers & HTTP

- [ ] Input validated before business logic
- [ ] Protected routes still guarded by auth/tenant/subscription middleware as appropriate
- [ ] Errors mapped to correct HTTP status; no raw internal errors to clients

## Errors & logging

- [ ] Uses `internal/errors` / wrapping with `%w` where appropriate
- [ ] Structured logging (not `fmt.Printf`)
- [ ] No secrets/tokens/passwords in logs

## Data & migrations

- [ ] Schema changes include migrations
- [ ] Multi-write paths use transactions when needed
- [ ] Tenant-scoped queries preserve isolation
- [ ] History tables updated when the existing domain pattern requires it

## Tests

- [ ] New behavior has unit and/or integration tests
- [ ] Test names follow `Test[Struct]_[Method]_[Scenario]`
- [ ] Error and edge paths covered for critical logic

## Security & config

- [ ] No hardcoded secrets
- [ ] Passwords hashed with bcrypt
- [ ] User-controlled input not trusted without validation
