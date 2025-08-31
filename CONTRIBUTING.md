Perfecto 🚀, te armo las reglas **ya pulidas y complementadas** para que queden listas como un **CONTRIBUTING.md**.

---

# 📖 Reglas de Contribución

Este documento define las reglas y estándares para contribuir al proyecto.
El objetivo es mantener **consistencia, calidad y mantenibilidad** en el código.

---

## 1. Arquitectura y Estructura

* Mantener el patrón **Clean Architecture**: `handlers → services → repository`.
* No crear dependencias cruzadas entre capas.
* Todas las dependencias externas deben ir detrás de una **interfaz**.
* Inyectar dependencias mediante constructores, nunca crear instancias directamente en la lógica.
* Seguir la estructura de carpetas actual:

  ```
  cmd/api/           → Punto de entrada
  internal/          → Código de aplicación
    ├── handlers/    → Controladores HTTP
    ├── services/    → Lógica de negocio
    ├── repository/  → Acceso a datos
    ├── middleware/  → Middlewares
    ├── errors/      → Manejo de errores
    └── validators/  → Validaciones
  model/             → Modelos de datos
  migrations/        → Migraciones de base de datos
  tests/             → Pruebas de integración
  ```

---

## 2. Nomenclatura y Estilo

* **PascalCase** para tipos y funciones públicas.
* **camelCase** para variables y funciones privadas.
* **snake\_case** en base de datos.
* Prefijos descriptivos en columnas de DB: `id_`, `created_on`, `modified_on`.
* Archivos: `[domain].go`, `[domain]_test.go`.
* Código en inglés (nombres, comentarios, docs).

---

## 3. Manejo de Errores

* Usar errores personalizados de `internal/errors/`.
* Hacer **wrapping de errores** con contexto:

  ```go
  return fmt.Errorf("failed to create order: %w", err)
  ```
* En los handlers: verificar si el error es un `HTTPError` y responder en consecuencia.
* Todo error importante debe registrarse con logging.

---

## 4. Logging

* No usar `fmt.Printf` para logs.
* Usar un logger estructurado (ej. `zerolog` o `zap`).
* Incluir contexto relevante: `user_id`, `order_id`, operación ejecutada.
* Usar niveles de log: `debug`, `info`, `warn`, `error`.

---

## 5. Testing

* **Unit tests** para repositorios con `sqlmock`.
* **Integration tests** con `testcontainers` y MySQL real.
* Naming: `Test[Struct]_[Method]_[Scenario]`.
* Usar `testify/assert` o `testify/require`.
* Deben cubrir: casos felices, casos edge y errores.
* CI/CD (GitHub Actions) debe correr todos los tests y pasar antes de mergear.

---

## 6. Base de Datos

* Todo cambio en DB debe hacerse vía **migraciones** (`golang-migrate`).
* Usar transacciones para operaciones complejas.
* Mantener tablas de historial (`products_history`, `orders_history`).
* Validar constraints también a nivel de aplicación.

---

## 7. Seguridad

* Toda entrada de usuario debe validarse en `validators/`.
* Rutas protegidas deben usar middleware de autenticación (JWT).
* Contraseñas siempre con hashing seguro (`bcrypt`).
* Sanitizar datos antes de almacenarlos.

---

## 8. Configuración

* Configuración vía variables de entorno (`.env`).
* Validar la configuración al inicio (`config.Validate()`).
* No hardcodear valores sensibles.
* Compatible con Docker y docker-compose.

---

## 9. Documentación

* Documentar funciones públicas e interfaces con comentarios en inglés.
* Mantener documentación de API (ej. Swagger/OpenAPI si aplica).
* Escribir **README claro** de cómo correr el proyecto en local.

---

## 10. Performance y Mantenibilidad

* Evitar N+1 queries en repositorios.
* Crear índices adecuados en la base de datos.
* Mantener funciones pequeñas y con responsabilidad única.
* Refactorizar código duplicado o confuso antes de agregar nuevas features.

---

✅ Con estas reglas, cualquier contribución (humana o asistida por IA) seguirá la misma línea de calidad, consistencia y escalabilidad.

---

¿Quieres que te lo deje directamente en formato **`CONTRIBUTING.md` listo para pegar en tu repo**, o prefieres mantenerlo como documento interno por ahora?
