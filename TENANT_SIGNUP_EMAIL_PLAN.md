# Tenant Signup Code — Email Delivery (MVP)

## Goal

Close the gap between **creating a signup code** and **tenant registration**.

Today the superadmin creates a one-time code and gets it in the API response; delivery to the recipient is manual. For MVP we email the code via Brevo. The frontend already owns the registration form fields; the backend only needs to send the code and keep validating it on `POST /public/tenant-register`.

## Target flow

```
Superadmin                          Backend                         Recipient                         Frontend
    |                                  |                                |                                 |
    | POST /auth/internal/             |                                |                                 |
    |   tenant-signup-codes            |                                |                                 |
    | { email, expires_in_minutes,     |                                |                                 |
    |   notes }                        |                                |                                 |
    |--------------------------------->|                                |                                 |
    |                                  | create OTC (hash stored)       |                                 |
    |                                  | store recipient email          |                                 |
    |                                  | Brevo: email with code (+link) |                                 |
    |                                  |------------------------------->|                                 |
    | 201 { id, expires_at, ... }      |                                | opens email / link               |
    |<---------------------------------|                                |-------------------------------->|
    |                                  |                                | fills tenant_name, slug,        |
    |                                  |                                | admin_name, email, phone,       |
    |                                  |                                | password, one_time_code         |
    |                                  |         POST /public/          |                                 |
    |                                  |           tenant-register      |                                 |
    |                                  |<---------------------------------------------------------------|
    |                                  | validate code, create tenant   |                                 |
    |                                  | + admin, consume code, JWT     |                                 |
    |                                  |--------------------------------------------------------------->|
```

## MVP scope (keep it simple)

**In scope**
- Require recipient `email` when creating a signup code.
- Persist that email on the signup code row.
- Send Brevo email containing the plaintext one-time code.
- Optional deep link to the frontend register page with `?code=...` (and optionally `?email=...`) so the form can prefill the code field only.
- Keep existing `POST /public/tenant-register` contract; still validates `one_time_code` as today.
- Frontend collects all registration fields and posts them back.

**Out of scope (post-MVP)**
- Prefilling tenant name / slug from the email.
- Binding registration so only the invited email can register (strict email match on consume).
- Resend / revoke UX beyond what already exists.
- Fancy email templates / Brevo template IDs.
- Unifying this with tenant user invitations (different product flow).

## Current building blocks to reuse

| Piece | Where |
|-------|--------|
| Create signup code (superadmin only) | `POST /auth/internal/tenant-signup-codes` |
| Public register with OTC | `POST /public/tenant-register` |
| OTC generate + hash | `TenantSignupService` + `AuthService.GenerateOneTimeToken` / `HashOneTimeToken` |
| Email sender (Brevo + noop) | `internal/services/email` |
| App base URL for links | `APP_BASE_URL` (same as invitations / password reset) |
| Table | `tenant_signup_codes` (already has `used_email` for post-consume; add recipient column for invite target) |

## API changes

### 1. `POST /auth/internal/tenant-signup-codes` (auth: superadmin)

**Request (updated)**

```json
{
  "email": "ana@panaderia.com",
  "expires_in_minutes": 120,
  "notes": "Cliente Panadería Sol"
}
```

- `email` — **required**, validated.
- `expires_in_minutes` — optional (default 120).
- `notes` — optional.

**Response (MVP)**

```json
{
  "id": 42,
  "code": "<plaintext one-time code>",
  "expires_at": "2026-07-21T20:00:00Z",
  "email": "ana@panaderia.com",
  "message": "Signup code sent successfully"
}
```

Notes:
- **Keep returning plaintext `code`** in the API response (same as today), as a fallback if email delivery is delayed or for manual/ops use. Primary delivery channel is still Brevo.
- If Brevo fails, fail the request (do not leave an orphaned usable code without notifying the recipient), matching invitation behavior.

### 2. `POST /public/tenant-register` (unchanged for MVP)

```json
{
  "tenant_name": "Panadería Sol",
  "tenant_slug": "panaderia-sol",
  "admin_name": "Ana Pérez",
  "email": "ana@panaderia.com",
  "phone": "0412-0000000",
  "password": "Secure1!",
  "one_time_code": "<code from email>"
}
```

Frontend owns all fields. Backend keeps validating the code as today.

## Email content (Brevo)

New method on `email.Sender`, e.g. `SendTenantSignupCode`:

- **To:** recipient email from create-code request.
- **Subject:** e.g. `Your bakery signup code`
- **Body (MVP):**
  - Plaintext one-time code (required).
  - Register link: `{APP_BASE_URL}/tenant-register?code={code}`
  - Short expiry note.

Config required (already used elsewhere):
- `BREVO_API_KEY` / from-email / from-name (existing Brevo wiring)
- `APP_BASE_URL` (for the optional link)

## Database

Add a migration, e.g. `000042_tenant_signup_codes_recipient_email`:

- Column `recipient_email VARCHAR(320) NOT NULL` (or nullable first if you need a soft rollout; for MVP greenfield local DBs, `NOT NULL` is fine if create always sends email).
- Optional index on `recipient_email` for support queries.
- Do **not** reuse `used_email` for the invite target — that column is for “email that consumed the code”.

## Implementation tasks

### Backend

- [x] **Migration:** add `recipient_email` to `tenant_signup_codes` (+ down migration).
- [x] **Model:** update `CreateSignupCodeRequest` / response (require `email`; keep plaintext `code` in response).
- [x] **Repository:** persist `recipient_email` on insert.
- [x] **Email package:**
  - [x] Add `TenantSignupCodePayload` (`ToEmail`, `Code`, `RegisterURL` optional).
  - [x] Add `SendTenantSignupCode` to `Sender` interface.
  - [x] Implement on `BrevoSender` and `NoopSender`.
- [x] **TenantSignupService.CreateSignupCode:**
  - [x] Validate email.
  - [x] Generate OTC + store hash + recipient email (existing TTL/notes).
  - [x] Build optional register URL from `APP_BASE_URL`.
  - [x] Call Brevo; on failure return error (and consider not committing / or mark revoked — prefer single transaction: insert then send; if send fails, revoke/delete or return error after insert with compensating revoke).
  - [x] Wire `EmailSender` + `AppBaseURL` into `TenantSignupService` in `cmd/api/main.go` (same pattern as invitations).
- [x] **Handler:** validate required email; map email/send errors to proper HTTP statuses.
- [x] **Security:** never log plaintext code at info; debug-only if needed. Never log passwords.

### Tests

- [x] Unit: create code requires email; forbidden for non-superadmin (existing).
- [x] Unit/service: successful create calls email sender with recipient + code.
- [x] Unit/service: Brevo failure → error response (no “silent success”).
- [x] Integration: create code as superadmin → register with emailed code still works (can use `NoopSender` and capture code via test double that records the payload).
- [x] Regression: `POST /public/tenant-register` unchanged behavior for valid/invalid/expired/used codes.

### Frontend (separate repo / checklist for FE)

- [ ] Register page at `/tenant-register` that accepts `?code=` (and optionally `?email=`) query params to prefill **only** those fields.
- [ ] Form fields: `tenant_name`, `tenant_slug`, `admin_name`, `email`, `phone`, `password`, `one_time_code`.
- [ ] Submit to `POST /public/tenant-register`.
- [ ] Handle 422 invalid code, 409 slug/email conflict, success → store JWT / redirect.

### Ops / config

- [ ] Ensure local `.env` has Brevo vars + `APP_BASE_URL` pointing at the admin frontend origin.
- [ ] Document in `readme.md` or `DEPLOY.md`: superadmin create-code now emails the recipient; no manual code copy required for happy path.

## Suggested implementation order

1. Migration + model/request changes.
2. Email interface + Brevo/noop implementations.
3. Service wiring (create → persist → send).
4. Handler + main.go DI.
5. Tests with a recording/fake sender.
6. FE deep-link prefill (can ship in parallel after email body includes the raw code).

## Acceptance criteria

1. Superadmin calls create-code with a recipient email and receives success **without needing to copy a code manually** for the happy path.
2. Recipient receives a Brevo email containing the one-time code.
3. Recipient (or QA) can complete registration via frontend using that code; backend validates and consumes the code as today.
4. Invalid/expired/used codes still rejected.
5. Non-superadmin still cannot create codes (`403`).

## Decisions

1. **Return plaintext code in API response?** **Yes** — keep returning `code` (fallback / ops), and also email it via Brevo.
2. **Frontend path for the link?** **`/tenant-register`** — email link: `{APP_BASE_URL}/tenant-register?code={code}`.
3. **Strict email binding?** MVP: **no** (send code to email; register may use any email). Post-MVP: require `PublicTenantRegisterRequest.email` to match `recipient_email`.
