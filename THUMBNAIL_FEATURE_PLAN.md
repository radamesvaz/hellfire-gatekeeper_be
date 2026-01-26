## Selección de thumbnail por producto

Guía para habilitar que un admin seleccione la imagen thumbnail de un producto.

### Backend
- **Migración**: agregar columna `thumbnail_url` a `products` y `product_history` (nullable); poblar con la primera `image_urls` existente para productos ya creados. Incluir down.
- **Modelos**: añadir `ThumbnailURL` a `Product`, `ProductHistory`, `CreateProductRequest`, `UpdateProductRequest` y a las respuestas JSON.
- **Repositorio**:
  - Incluir `thumbnail_url` en los `SELECT/INSERT/UPDATE`.
  - Nuevo método `UpdateProductThumbnail(id, thumbnailURL string)` que valide que `thumbnailURL` pertenece a `image_urls`; error 400 si no pertenece.
  - En `UpdateProductImages`, si no hay thumbnail asignado, opcionalmente setear el primero como default.
- **Handlers**:
  - Exponer `thumbnail_url` en `GetAllProducts` y `GetProductByID`.
  - Nuevo endpoint admin `PATCH /products/{id}/thumbnail` con body `{ "thumbnail_url": "<url>" }`:
    - Carga producto, valida pertenencia, actualiza repo, registra histórico.
  - En `DeleteProductImage`: si la URL eliminada es el thumbnail, decidir política (limpiar o reasignar al primer restante) y reflejarlo en histórico.
  - En `ReplaceProductImages`/`AddProductImages`: si no hay thumbnail, setear el primero de la lista resultante.
- **Histórico**: propagar `ThumbnailURL` en `UpdateHistoryTable` tanto de `ProductHandler` como de `ImageHandler`.
- **Tests**: cubrir validación de pertenencia, reasignación al borrar, endpoint nuevo y respuestas JSON con `thumbnail_url`.

### Frontend
- **Modelo/estado**: añadir `thumbnail_url` al tipo Producto y consumirlo en listados/detalles.
- **UI admin**:
  - En galería de imágenes mostrar acción “Usar como thumbnail”.
  - Marcar visualmente la imagen seleccionada; deshabilitar acción si ya es thumbnail.
  - Al eliminar una imagen, refrescar y mostrar la nueva selección si cambió.
- **API calls**:
  - Usar `PATCH /products/{id}/thumbnail` con `{ thumbnail_url }`.
  - Manejar errores 400/404 (URL no existente o producto no encontrado).
- **Render público**: usar `thumbnail_url` como principal; si viene vacío, fallback a `image_urls[0]`.
- **Tests/QA**: flujos de selección, fallback en catálogo, borrado de imagen que era thumbnail y manejo de errores.
