# Frontend changelog — Product flow hardening

Backend changes the frontend must adopt. Use this as the migration checklist for catalog, admin product management, and checkout.

---

## Summary for FE

| Area | What changed |
|------|----------------|
| Product field `available` | **Removed.** Do not send or read it. |
| Product field `track_inventory` | **Added.** Controls whether stock is enforced. |
| Public catalog | Only returns `status: "active"` products. |
| Admin catalog | New authenticated endpoints return **all** statuses. |
| Create product | Returns **201**; validation stricter; images still separate. |
| Update product (`PUT`) | No longer accepts `image_urls` / `thumbnail_url`. |
| Checkout / create order | New HTTP status mapping (404 / 409); duplicate line items merged server-side. |
| Auth on product mutations | Admin or superadmin JWT required (403 otherwise). |
| Tenant context | Required; missing tenant → **400** `tenant context required`. |

---

## 1. Product model (API shape)

### Removed

```json
"available": true
```

Delete any toggle, type field, form control, or filter based on `available`.

### Added

```json
"track_inventory": true
```

| Value | Meaning for UI |
|-------|----------------|
| `true` (default) | Stock is tracked. Show stock field; block buy / show “Agotado” when `stock === 0`. |
| `false` | Unlimited inventory. Stock is ignored for purchase. Hide or disable stock as a purchase gate; optional to still show stock for internal notes. |

### Unchanged (but now enforced)

```json
"status": "active" | "inactive" | "deleted"
"stock": 0
```

| `status` | Public storefront | Purchasable | Admin |
|----------|-------------------|-------------|--------|
| `active` | Visible | Yes (if stock allows) | Manage |
| `inactive` | Hidden | No | Keep / reactivate |
| `deleted` | Hidden | No | Soft-deleted |

### Example product JSON (response)

```json
{
  "id_product": 12,
  "tenant_id": 1,
  "name": "Brownie",
  "description": "Clásico",
  "price": 3.5,
  "track_inventory": true,
  "stock": 8,
  "status": "active",
  "image_urls": ["https://res.cloudinary.com/.../main.jpg"],
  "thumbnail_url": "https://res.cloudinary.com/.../main.jpg",
  "created_on": "2026-07-23T10:00:00Z"
}
```

---

## 2. Public catalog (customer-facing)

### Endpoints

- `GET /t/{tenant_slug}/products`
- `GET /t/{tenant_slug}/products/{id}`
- Legacy `GET /products` / `GET /products/{id}` (prefer tenant path)

### Behavior changes

1. **List / get only return `status === "active"`.**
   - Inactive / deleted products no longer appear.
   - `GET .../products/{id}` for inactive/deleted → **404** `Product not found`.
2. Still supports pagination: `limit`, `cursor`, optional `q` (name contains, min 2 chars).
3. Tenant must be resolved (path slug or middleware). Missing tenant → **400**.

### FE actions

- Stop filtering by `available` on the client.
- For buy button / cart:
  - If `track_inventory === false` → always allow add-to-cart (while product is shown).
  - If `track_inventory === true` and `stock === 0` → show sold out; do not add to cart.
  - Prefer showing sold-out on active products rather than hiding them (backend still lists `stock: 0` active products).

---

## 3. Admin product endpoints (authenticated)

All under `/auth/...` with:

- `Authorization: Bearer <jwt>`
- Tenant middleware (JWT tenant / path as today)
- **Role: admin (`role_id = 1`) or superadmin (`role_id = 3`)**  
  Client role → **403 Forbidden**

### New / clarified reads (all statuses)

| Method | Path | Behavior |
|--------|------|----------|
| `GET` | `/auth/products` | List products for the tenant (**all statuses**: active, inactive, deleted). Same pagination query params as public list. |
| `GET` | `/auth/products/{id}` | Get one product **regardless of status**. |

Use these for the admin catalog UI. Do **not** use public `GET /t/.../products` to manage inactive/deleted items.

### Create

`POST /auth/products`

**Body (JSON):**

```json
{
  "name": "Brownie",
  "description": "Clásico",
  "price": 3.5,
  "stock": 10,
  "track_inventory": true,
  "status": "active"
}
```

| Field | Rules |
|-------|--------|
| `name` | Required, non-empty |
| `description` | Required, non-empty |
| `price` | Required conceptually; must be `>= 0` (free products allowed) |
| `stock` | Optional; default `0` |
| `track_inventory` | Optional; **omit or null → `true`** |
| `status` | Optional; omit → `"active"`. Must be `active` \| `inactive` \| `deleted` |
| `available` | **Do not send** |
| `image_urls` | **Do not send** on create — gallery starts empty |

**Success:** **201 Created**

```json
{
  "message": "Product created successfully",
  "product_id": 12,
  "image_urls": []
}
```

After create, upload images via image endpoints (below).

### Update fields

`PUT /auth/products/{id}`

**Body (JSON):**

```json
{
  "name": "Brownie",
  "description": "Clásico",
  "price": 3.5,
  "stock": 10,
  "track_inventory": true,
  "status": "active"
}
```

| Field | Notes |
|-------|--------|
| `name`, `description` | Required |
| `price` | `>= 0` |
| `track_inventory` | Optional pointer: **omit keeps current value**. Send `true`/`false` to change. |
| `status` | Optional: omit keeps current. |
| `image_urls` | **Removed from this contract — ignored / not accepted for gallery updates** |
| `thumbnail_url` | **Not on this PUT** — use thumbnail endpoints |
| `available` | **Removed** |

**Success:** 200

### Soft delete / pause

`PATCH /auth/products/{id}`

```json
{ "status": "inactive" }
```

or

```json
{ "status": "deleted" }
```

| Status | FE use |
|--------|--------|
| `inactive` | Pause without deleting (hide from storefront) |
| `deleted` | Soft delete |
| `active` | Publish / restore |

On `deleted`, backend best-effort cleans local image files. Cloudinary URLs may still exist until purged separately; UI should treat product as gone from storefront immediately.

### Images & thumbnails (unchanged routes, still the only gallery path)

| Method | Path | Purpose |
|--------|------|---------|
| `POST` | `/auth/products/{id}/images` | Add images (`multipart`, field `images`) |
| `PUT` | `/auth/products/{id}/images` | Replace all images (`multipart`, field `images`) |
| `DELETE` | `/auth/products/{id}/images?imageUrl=...` | Delete one image |
| `PATCH` | `/auth/products/{id}/thumbnail` | Set thumbnail URL (must already be in `image_urls`) |
| `POST` | `/auth/products/{id}/thumbnail` | Upload dedicated thumbnail (`multipart`, field `thumbnail`) |

**Limits:** max **5MB** per image/thumbnail; types jpeg/jpg/png/webp.

**Do not** send gallery URLs on `PUT /auth/products/{id}` — that never persisted and is no longer part of the update model.

---

## 4. Suggested admin UI controls

Replace the old “Available” toggle with:

1. **Status** — Active / Inactive / Deleted (or soft-delete action)
2. **Track inventory** — checkbox/switch (“Limitar stock online”)
3. **Stock** — number input; enabled mainly when track inventory is on

Mental model for merchants:

- Pause selling without deleting → `inactive`
- Sold out but still listed → `active` + `track_inventory: true` + `stock: 0`
- Don’t want to babysit stock → `track_inventory: false`

---

## 5. Create order (checkout)

### Endpoints

- Prefer: `POST /t/{tenant_slug}/orders`
- Legacy: `POST /orders` (still needs tenant in context; prefer path-based)

### Purchase rules (server-enforced)

A line item succeeds only if:

```text
product exists for tenant
AND status == "active"
AND (track_inventory == false OR stock >= quantity)
```

Duplicate `id_product` lines in the same payload are **merged** (quantities summed) before stock checks.

### HTTP status changes (important)

| Situation | Old (typical) | New |
|-----------|---------------|-----|
| Product missing | Often 500 | **404** |
| Product inactive / deleted | Often 500 / still buyable | **409** `product not available for purchase` |
| Not enough stock | Often 500 | **409** `not enough product stock` |
| Missing tenant | Silent tenant `1` | **400** `tenant context required` |
| Success | 200 | 200 (unchanged) |

### FE actions

- Map **409** to user-friendly messages (sold out / product unavailable).
- Map **404** to “product no longer available”.
- Refresh cart / catalog after 409.
- Stop relying on `available` for checkout eligibility.
- Optional client-side pre-check using `status` + `track_inventory` + `stock` to reduce 409s.

### Order status update (admin cancel)

Cancelling an already cancelled / expired order is rejected (invalid transition). Stock is reverted once, only for products with `track_inventory: true`.

---

## 6. Auth & tenant checklist

1. Product **mutations** and **admin product reads** (`/auth/products*`) need admin/superadmin token.
2. Always send requests in a tenant-scoped way (`/t/{slug}/...` or authenticated tenant context). Do not assume default tenant `1`.
3. Client-role users must not call admin product APIs (expect 403).

---

## 7. Types / codegen notes

Suggested TypeScript shape:

```ts
type ProductStatus = "active" | "inactive" | "deleted";

interface Product {
  id_product: number;
  tenant_id: number;
  name: string;
  description: string;
  price: number;
  track_inventory: boolean;
  stock: number;
  status: ProductStatus;
  image_urls: string[];
  thumbnail_url: string;
  created_on?: string | null;
}

interface CreateProductRequest {
  name: string;
  description: string;
  price: number;
  stock?: number;
  track_inventory?: boolean; // omit => true on server
  status?: ProductStatus;    // omit => "active"
}

interface UpdateProductRequest {
  name: string;
  description: string;
  price: number;
  stock: number;
  track_inventory?: boolean; // omit => keep existing
  status?: ProductStatus;    // omit => keep existing
}
```

Remove `available` from all Product types and forms.

Helpers:

```ts
function isPurchasable(p: Product): boolean {
  if (p.status !== "active") return false;
  if (!p.track_inventory) return true;
  return p.stock > 0;
}

function isSoldOut(p: Product): boolean {
  return p.status === "active" && p.track_inventory && p.stock === 0;
}
```

---

## 8. Migration checklist (FE)

- [ ] Remove `available` from types, forms, filters, and API payloads
- [ ] Add `track_inventory` to product form (default ON)
- [ ] Admin: use `GET /auth/products` (+ optional `GET /auth/products/{id}`) for management
- [ ] Storefront: use public tenant product endpoints only
- [ ] Create product: expect **201**; upload images after create
- [ ] Update product: stop sending `image_urls` / `thumbnail_url` on PUT
- [ ] Soft delete / pause via `PATCH` status
- [ ] Checkout: handle **404** and **409** from create order
- [ ] Sold-out UI based on `track_inventory` + `stock`, not `available`
- [ ] Ensure product admin calls use admin JWT (not client)

---

## 9. Out of scope / unchanged for FE

- Image upload still multipart to the same image routes (Cloudinary URL or `/uploads/...` depending on backend env).
- Order create success response shape unchanged (`{ "message": "Order created successfully" }`).
- Soft-deleted products are not hard-deleted; pending orders are not auto-cancelled when a product is soft-deleted.
