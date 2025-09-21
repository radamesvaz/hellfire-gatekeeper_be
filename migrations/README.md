
---

# 📦 Database Migrations

Este proyecto usa [`golang-migrate`](https://github.com/golang-migrate/migrate) para manejar cambios en el esquema de la base de datos de forma segura y estructurada.

## 🚀 Aplicar Migraciones

### Opción 1: Scripts Automáticos (Recomendado)

```bash
./run.sh migrate
```

### Opción 2: Comando Manual

```bash
go run cmd/migrate/main.go up
```

### Opción 3: Con Docker

```bash
docker-compose up migrate
```

## 🛠 Crear Nueva Migración

Cada migración consiste en **dos archivos**:
- Uno para aplicar los cambios (`up`)
- Uno para revertirlos (`down`)

Para crear una migración (ejemplo: agregar tabla `roles`):

```bash
migrate create -ext sql -dir migrations -seq add_roles_table_and_user_relation
```

Esto generará:

```
migrations/
  ├── 000001_add_roles_table_and_user_relation.up.sql
  └── 000001_add_roles_table_and_user_relation.down.sql
```

### Ejemplo: `000001_add_roles_table_and_user_relation.up.sql`

```sql
CREATE TABLE roles (
  id_role SERIAL PRIMARY KEY,
  name VARCHAR(50) NOT NULL UNIQUE
);

INSERT INTO roles (name) VALUES ('admin'), ('client');

ALTER TABLE users
ADD COLUMN id_role INT NOT NULL DEFAULT 2,
ADD CONSTRAINT fk_role FOREIGN KEY (id_role) REFERENCES roles(id_role);
```

### Ejemplo: `000001_add_roles_table_and_user_relation.down.sql`

```sql
ALTER TABLE users DROP CONSTRAINT fk_role;
ALTER TABLE users DROP COLUMN id_role;

DROP TABLE roles;
```

## 🔁 Revertir Migraciones

Para deshacer la última migración aplicada:

```bash
go run cmd/migrate/main.go down
```

## 📋 Migraciones Disponibles

1. **000001** - Esquema inicial (usuarios, productos, órdenes)
2. **000002** - Tabla de roles y relación con usuarios
3. **000003** - Datos de prueba (seed data)
4. **000004** - Columna de estado para productos
5. **000005** - Clave foránea en historial de productos
6. **000006** - Enum en historial de productos
7. **000007** - Columna de stock en productos
8. **000008** - Valores de stock en productos
9. **000009** - Columna de nota y tiempo en órdenes
10. **000010** - Datos mock de órdenes
11. **000011** - Enum de acción 'create' en historial de órdenes
12. **000012** - URLs de imágenes en productos
13. **000013** - Columna 'paid' en órdenes
14. **000014** - Actualizar datos mock con estado 'paid'
15. **000015** - Corregir secuencias
16. **000016** - Agregar estado 'deleted' a órdenes

---

**Handling Migration Failures with GitHub Actions**

When working with database migrations, it's crucial to detect errors early and ensure consistency across environments. Thanks to the automated testing pipeline we implemented in GitHub Actions, any issues introduced by faulty migrations will be caught automatically before merging changes to the `master` branch.

### How It Works

Our GitHub Actions workflow is configured to:
1. Set up the test environment (including creating `.env` with test DB credentials).
2. Apply migrations using `golang-migrate`.
3. Run the complete test suite.

If any migration causes errors (e.g., syntax errors, constraint issues, missing fields), the test suite will fail, and the workflow will report a failed check on the Pull Request. This prevents the merge from happening unless the issue is resolved.

### Why This Helps
- **Prevents breaking `master`**: Ensures that only verified migrations reach the main branch.
- **Saves debugging time**: Issues are caught automatically before deployment.
- **Keeps development safe**: Team members won’t pull a broken schema.

### How to Recover
If a migration causes the workflow to fail:
1. Fix the migration file or create a new one that reverts the change.
2. Push the fix to your feature branch.
3. Let GitHub Actions rerun the workflow.

Once the check passes, you'll be able to merge safely.

### Best Practice
- Always run migrations locally before committing.
- Keep migration files atomic and minimal.
- Include tests that confirm the presence of required tables, fields, or constraints.

By leveraging the GitHub Actions pipeline, we build confidence in our database layer and avoid accidental downtime caused by unchecked schema changes.
