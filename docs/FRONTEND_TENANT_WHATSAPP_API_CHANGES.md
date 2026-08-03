# Frontend changelog — Tenant WhatsApp phone

Backend changes the frontend must adopt so checkout invoice deep links and admin settings use the bakery’s WhatsApp number from the API (not a hardcoded / env-only value on the client).

Design context: [`TENANT_WHATSAPP_PHONE.md`](./TENANT_WHATSAPP_PHONE.md).

---

## Summary

| Area | What changed |
|------|----------------|
| Store WhatsApp contact | **New** tenant field `whatsapp_phone` (not the admin user’s `phone`) |
| Public branding | `GET /t/{slug}/branding` → `branding.whatsapp_phone` |
| Admin update | `PATCH /auth/branding/whatsapp` (admin JWT) |
| Tenant register | Optional `whatsapp_phone` on `POST /public/tenant-register` |
| Backend WhatsApp send | **None** — FE still opens WhatsApp / `wa.me` |

---

## 1. Mental model (do not mix these)

| Field | Where it lives | Use for |
|-------|----------------|---------|
| `phone` on register / user | Admin `users.phone` | Account / contact for the person who signed up |
| `whatsapp_phone` | `tenants.whatsapp_phone` | Storefront invoice destination (“message the bakery”) |

Same person often owns both at signup. The API does **not** copy admin `phone` into `whatsapp_phone`. The registration UI may prefill WhatsApp from the admin phone for convenience, but must send `whatsapp_phone` explicitly if it should be stored.

---

## 2. Public branding (storefront)

### Endpoint

`GET /t/{tenant_slug}/branding`  
(or legacy path + `X-Tenant-Slug` if you still use it)

### Response shape (relevant part)

```json
{
  "tenant_id": 1,
  "tenant_slug": "acme-bakery",
  "branding": {
    "tenant_name": "Acme Bakery",
    "logo_url": "https://...",
    "primary_color": "#111827",
    "secondary_color": "#374151",
    "accent_color": "#F59E0B",
    "whatsapp_phone": "+584121234567"
  }
}
```

| Value | Meaning |
|-------|---------|
| Non-empty string | Use for checkout → WhatsApp invoice link |
| `""` | Unset — do not invent a number; warn / block “send invoice” |

Types / store: add `whatsapp_phone: string` on the branding object (default `""` if missing for older caches).

### Checkout

1. Load branding (you likely already do for name/colors/logo).
2. After a successful order create (`201` + `id_order`), build the WhatsApp link from `branding.whatsapp_phone`.
3. If `whatsapp_phone` is empty, show a clear message (“Bakery WhatsApp not configured”) instead of opening WhatsApp with a wrong/hardcoded number.

Suggested link shape (FE responsibility):

```text
https://wa.me/<digitsOnly>?text=<urlencoded invoice message>
```

Strip non-digits for `wa.me` if your current helper already does that; keep displaying the stored string as-is in admin UI.

---

## 3. Update WhatsApp (admin settings / branding)

### Endpoint

`PATCH /auth/branding/whatsapp`

- Auth: **admin JWT** (same tenant context as other `/auth/branding/*` routes)
- Client role → **403 Forbidden**
- Superadmin is not treated as admin on this route → **403** (use a tenant admin session)

### Request

```json
{ "whatsapp_phone": "+584121234567" }
```

| Case | Behavior |
|------|----------|
| `"whatsapp_phone": "+58..."` | Set / replace |
| `"whatsapp_phone": ""` | Clear (stored as unset; branding returns `""`) |
| Key omitted | **400** `whatsapp_phone is required` |
| Longer than **20** characters (after trim) | **400** |
| Invalid JSON | **400** |

### Success (200)

```json
{
  "message": "WhatsApp phone updated successfully",
  "tenant_id": 1,
  "tenant_slug": "acme-bakery",
  "whatsapp_phone": "+584121234567"
}
```

After a successful PATCH, update local branding state (or refetch `GET .../branding`) so the storefront preview matches.

---

## 4. Tenant registration

### Endpoint

`POST /public/tenant-register`

### Body (new optional field)

```json
{
  "tenant_name": "Acme Bakery",
  "admin_name": "Owner Admin",
  "email": "owner@acme.com",
  "phone": "555-0101",
  "whatsapp_phone": "+584121234567",
  "password": "StrongPass123!",
  "one_time_code": "ABCD1234-EFGH5678"
}
```

| Field | Required | Stored as |
|-------|----------|-----------|
| `phone` | optional (unchanged) | Admin user phone |
| `whatsapp_phone` | optional | Tenant store WhatsApp |

- Omitted or blank `whatsapp_phone` → tenant WhatsApp left unset (`""` on branding until configured).
- Max length **20** after trim → **400** if exceeded.
- Register response does **not** need to return `whatsapp_phone`; confirm via branding after login if needed.

### UX suggestion

- Label admin phone vs store WhatsApp clearly.
- Prefill WhatsApp from admin phone in the form only; still submit `whatsapp_phone` as its own field.
- If left empty at signup, prompt in admin branding/settings before the bakery goes live.

---

## 5. Existing tenants

Tenants created before this change have `whatsapp_phone` empty until an admin sets it.

Do **not** fall back to:

- Hardcoded FE env WhatsApp
- Admin user phone from login/profile (unless you explicitly choose that as a temporary UX fallback — backend will not)

Preferred: empty state + CTA to “Set bakery WhatsApp” in branding settings.

---

## 6. Migration checklist (FE)

- [ ] Extend branding type / store with `whatsapp_phone: string`
- [ ] Checkout invoice link reads `branding.whatsapp_phone` only
- [ ] Handle empty WhatsApp (warn / block), no silent hardcoded fallback
- [ ] Registration: optional store WhatsApp field; map to `whatsapp_phone` (not `phone`)
- [ ] Admin branding/settings: display + edit via `PATCH /auth/branding/whatsapp`
- [ ] Allow clear (send `""`) if product supports “remove number”
- [ ] Always send the `whatsapp_phone` key on PATCH (never omit)
- [ ] Enforce max 20 chars in the form to match API

---

## 7. Out of scope (no FE work expected from backend)

- Backend sending WhatsApp / invoice templates
- Dedicated public “tenant details” endpoint (use branding)
- Bootstrap / invitation payloads including `whatsapp_phone`
