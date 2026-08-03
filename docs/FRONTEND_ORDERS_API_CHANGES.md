# Frontend changelog — Orders flow hardening

Backend changes the frontend must adopt for checkout and admin order management.

---

## Summary

| Area | What changed |
|------|----------------|
| `/auth/orders*` | **Admin / superadmin only** (client JWT → **403**) |
| `GET /auth/orders` sort | **Newest first** (`created_on DESC`); cursor **v3** (v2 invalid) |
| Status workflow | **Linear only:** `pending → preparing → ready → delivered` |
| Cancel | Only from `pending` / `preparing` / `ready` — **not** from `delivered` |
| Soft-delete | Only from `cancelled` / `expired` / `delivered` — never touches stock |
| Stock restore | On cancel (pre-delivery) and ghost `expired`; never after `delivered` |
| Subscription canceled | Public create → **404**; auth orders → **403** |
| Create success | **201** + `{ "message", "id_order" }` |
| Delivery date | **Today allowed** (calendar day ≥ today) |

---

## 1. Auth — who can call `/auth/orders*`

| Method | Path | Who |
|--------|------|-----|
| `GET` | `/auth/orders` | Admin / superadmin |
| `GET` | `/auth/orders/{id}` | Admin / superadmin |
| `PATCH` | `/auth/orders/{id}` | Admin / superadmin |

Client role → **403 Forbidden**.

Storefront checkout stays public (`POST /t/{slug}/orders` or legacy `POST /orders` + `X-Tenant-Slug`) — no JWT.

---

## 2. Status workflow (admin PATCH)

### Allowed

```text
pending → preparing → ready → delivered

pending | preparing | ready → cancelled   (restores tracked stock)

cancelled | expired | delivered → deleted   (no stock change)
```

### Rejected (400)

- Skips (`pending → ready`, `pending → delivered`, …)
- Backwards (`ready → preparing`, …)
- `delivered → cancelled`
- `pending → deleted` (and any delete before cancelled/expired/delivered)
- Changes from `cancelled` / `expired` / `deleted` (except `→ deleted` from cancelled/expired)

`expired` is set only by the ghost worker (unpaid timeout), not by PATCH.

### Stock

| Event | Tracked inventory |
|-------|-------------------|
| Create | Decrement |
| Cancel (pre-delivery) | Restore |
| Expire (worker) | Restore |
| Delivered | Keep decremented (sale) |
| Delete | No stock change |

---

## 3. Create order (public)

### Endpoints

- `POST /t/{tenant_slug}/orders` (preferred)
- `POST /orders` + `X-Tenant-Slug` (legacy)

### Success (new)

**201 Created**

```json
{
  "message": "Order created successfully",
  "id_order": 42
}
```

Use `id_order` for confirmation / deep links.

### Delivery date

- Format: `YYYY-MM-DD`
- **Today is valid**
- Past calendar days → 400

### Subscription canceled

Public create (path/header tenant resolution) returns **404**  
`tenant not found or inactive` — same as other public tenant routes when subscription is `canceled`.

Auth `/auth/orders*` with a canceled tenant returns **403**  
`tenant subscription is canceled`.

Do not allow checkout while the bakery is not paying.

### Other create errors (unchanged mapping)

| Case | HTTP |
|------|------|
| Validation | 400 |
| Product not found | 404 |
| Product not purchasable (inactive/deleted) | 409 |
| Not enough stock | 409 |

Duplicate `id_product` lines are still merged server-side.

---

## 4. Soft-delete vs cancel (admin UX)

- **Cancel** = stop the order before delivery → inventory comes back (if tracked).
- **Delete** = hide from default admin list **after** the order is already finished (`cancelled` / `expired` / `delivered`). No inventory side effects.

Default `GET /auth/orders` still hides `deleted` unless `ignore_status=true` or `status=deleted`.

---

## 4b. List sort — newest first

`GET /auth/orders` returns orders by **`created_on` DESC** (tie-break `id_order` DESC).

- First page = most recent orders (also when using `q` / `status` / `id_user`).
- Order cursor is **version 3**. Old v2 cursors are rejected (**400** `Invalid cursor`) — drop any cached `next_cursor` and refetch from page 1.

---

## 5. Migration checklist (FE)

- [ ] Ensure admin order screens use **admin JWT** (not client)
- [ ] Handle **403** on `/auth/orders*` for non-admin
- [ ] Status UI: only next linear step + cancel (pre-delivery) + delete (terminal only)
- [ ] Block cancel button on `delivered`
- [ ] Block delete until cancelled / expired / delivered
- [ ] Create order: expect **201** and read `id_order`
- [ ] Allow selecting **today** as delivery date
- [ ] Handle **404** on public create when subscription canceled (“store unavailable”)
- [ ] Sold-out / 409 handling unchanged from product hardening
- [ ] Admin order list: assume **newest first**; discard stale order cursors (v2 → v3)

---

## 6. Out of scope (no FE work expected yet)

- Customer self-service order portal
- `GET /auth/orders/{id}/history` endpoint
