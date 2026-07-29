# Tenant WhatsApp phone (MVP)

Storefront checkout sends the invoice to the bakery’s WhatsApp number. That number must live in the backend as tenant config, not only on the frontend.

---

## Decisions

| Decision | Choice | Why |
|----------|--------|-----|
| Where to store it | `tenants.whatsapp_phone` | Public storefront contact is tenant-level, not a user attribute |
| Separate from admin `phone` | Yes | Admin phone = signup/login contact; WhatsApp = invoice destination. Same person often owns both, but they can diverge |
| How FE reads it | Extend `GET /t/{slug}/branding` | Storefront already loads branding; no extra details endpoint for MVP |
| How FE updates it | `PATCH /auth/branding/whatsapp` (admin) | Same pattern as branding name/colors/logo |
| Set at register | Optional `whatsapp_phone` on `POST /public/tenant-register` | One-step onboarding; can still set later via branding PATCH |
| Copy admin phone → WhatsApp | No (API) | FE may prefill the form; backend stores intent only if sent |
| Backend WhatsApp send | Out of scope | FE keeps opening `wa.me` / WhatsApp client |
| Dedicated details endpoint | Deferred | Branding extension is enough for MVP |
| Existing tenants | `NULL` / empty until set | No backfill from admin user phone (ambiguous with multi-admin) |
| Validation (MVP) | Trim + max length 20 (match `users.phone`) | Avoid over-validating E.164 until we standardize phone formats |

---

## Tasks

### Backend

- [x] Document decisions and FE contract (this file)
- [x] Migration: add nullable `tenants.whatsapp_phone VARCHAR(20)`
- [x] Include `whatsapp_phone` in branding read (`TenantBranding` / `GetBranding`)
- [x] Admin `PATCH /auth/branding/whatsapp` to set or clear the number
- [x] Accept optional `whatsapp_phone` on `POST /public/tenant-register` and persist on tenant create
- [x] Unit tests (handlers / signup sqlmock expectations)
- [x] Targeted unit packages pass (`internal/handlers`, `auth`, `validators`)

### Frontend (follow-up)

- [ ] Read `branding.whatsapp_phone` for checkout invoice deep link
- [ ] Registration form: optional store WhatsApp field (can prefill from admin phone in UI only)
- [ ] Admin branding/settings: show + edit WhatsApp number
- [ ] Handle empty/missing number (block or warn before “send invoice”)

### Explicitly out of scope (MVP)

- Backend message/invoice sending
- New public “tenant details” endpoint
- Bootstrap / invitation flows gaining the field
- Migrating historical admin phones into `whatsapp_phone`

---

## API contract

### Public branding

`GET /t/{tenant_slug}/branding`

`branding` object gains:

```json
{
  "tenant_name": "...",
  "logo_url": "...",
  "primary_color": "...",
  "secondary_color": "...",
  "accent_color": "...",
  "whatsapp_phone": "+584121234567"
}
```

Unset → `""` (same null→empty pattern as `logo_url`).

### Update (admin JWT)

`PATCH /auth/branding/whatsapp`

```json
{ "whatsapp_phone": "+584121234567" }
```

- Empty string clears the number (`NULL` in DB).
- Non-admin → `403`.
- Longer than 20 chars → `400`.

### Register

`POST /public/tenant-register`

```json
{
  "tenant_name": "...",
  "admin_name": "...",
  "email": "...",
  "phone": "...",
  "whatsapp_phone": "...",
  "password": "...",
  "one_time_code": "..."
}
```

- `phone` → admin `users.phone` (unchanged)
- `whatsapp_phone` → `tenants.whatsapp_phone` (optional)
- Omitted / blank → tenant WhatsApp left unset
