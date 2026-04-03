# Changelog

## API (listados)

- **GET `/products`** y **GET `/t/{tenant_slug}/products`**: la respuesta es un objeto `{ "items", "next_cursor" }` (ya no un array en la raíz). Query opcional **`q`**: prefijo de nombre (insensible a mayúsculas), mínimo 2 caracteres; combinable con `limit` y `cursor`.
- **GET `/auth/orders`**: mismo envelope `{ "items", "next_cursor" }`. Query opcional **`id_user`**: filtra pedidos de ese usuario (entero `> 0`); combinable con `ignore_status`, `status`, `limit`, `cursor`.

Los cursores de productos y de órdenes **no** son intercambiables (formatos distintos).

**Documentación:** contrato OpenAPI 3 en **`docs/openapi.yaml`**; cómo visualizarlo y validarlo en el README raíz (sección *OpenAPI*).
