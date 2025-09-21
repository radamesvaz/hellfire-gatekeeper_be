
# 🚀 Hellfire Gatekeeper Backend

API backend para la aplicación de panadería desarrollada en Go con PostgreSQL.

## 🚀 Inicio Rápido

### Prerrequisitos

- Go 1.19+
- Docker y Docker Compose
- PostgreSQL (via Docker)

### Configuración

1. **Clona el repositorio:**
```bash
git clone <repository-url>
cd hellfire-gatekeeper_be
```

2. **Crea un archivo `.env` en la raíz del proyecto:**
```env
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres123
POSTGRES_DB=bakery_db
DB_HOST=localhost
DB_PORT=5432
JWT_SECRET=tu_jwt_secret_muy_seguro_aqui
JWT_EXPIRATION_MINUTES=60
PORT=8080
```

3. **Inicia el entorno de desarrollo:**
```bash
./run.sh dev
```

4. **Ejecuta la aplicación:**
```bash
./run.sh app
```

## 📋 Scripts de Desarrollo

### Scripts Principales

- **`./run.sh dev`** - Inicia PostgreSQL y ejecuta migraciones
- **`./run.sh app`** - Ejecuta la aplicación
- **`./run.sh migrate`** - Solo ejecuta migraciones
- **`./run.sh tests`** - Ejecuta todos los tests
- **`./run.sh unit`** - Ejecuta tests unitarios
- **`./run.sh integration`** - Ejecuta tests de integración
- **`./run.sh reset`** - Reinicia el proyecto completo

### Comandos Manuales

Si prefieres no usar los scripts:

```bash
# 1. Iniciar PostgreSQL
docker-compose up postgres_db -d

# 2. Ejecutar migraciones
go run cmd/migrate/main.go up

# 3. Ejecutar aplicación
go run cmd/api/main.go
```

## 🔧 Comandos Útiles

### Verificar PostgreSQL
```bash
docker ps
```

### Detener PostgreSQL
```bash
docker-compose down
```

### Ver logs de PostgreSQL
```bash
docker-compose logs postgres_db
```

## 📚 API Endpoints

### Productos
- `GET /products` - Obtener todos los productos
- `GET /products/{id}` - Obtener producto por ID
- `POST /auth/products` - Crear producto (requiere autenticación)
- `PUT /auth/products/{id}` - Actualizar producto (requiere autenticación)

### Órdenes
- `GET /auth/orders` - Obtener todas las órdenes (requiere autenticación)
- `GET /auth/orders?ignore_status=true` - Obtener todas las órdenes incluyendo eliminadas
- `GET /auth/orders?status=pending` - Filtrar órdenes por estado
- `GET /auth/orders/{id}` - Obtener orden por ID (requiere autenticación)
- `POST /auth/orders` - Crear orden (requiere autenticación)

### Autenticación
- `POST /login` - Iniciar sesión
- `POST /register` - Registrarse

## 🗄️ Base de Datos

Este proyecto usa **PostgreSQL** con migraciones automáticas. Las migraciones se encuentran en el directorio `migrations/`.

### Estados de Órdenes
- `pending` - Pendiente
- `preparing` - En preparación
- `ready` - Listo
- `delivered` - Entregado
- `cancelled` - Cancelado
- `deleted` - Eliminado

## 🧪 Testing

### Ejecutar Tests
```bash
go test ./...
```

### Tests con Docker
```bash
./run.sh tests
```

## ✅ Continuous Testing with GitHub Actions

This project includes an automated test workflow using **GitHub Actions**. Every time you push or open a pull request against the `master` branch, a CI pipeline runs to validate that all tests pass.

### What It Covers

- ✅ **Unit tests** (run locally using Go's testing framework)
- ✅ **Integration tests** (using `testcontainers-go` and Docker)
- ✅ **Migration safety** (any change that breaks the database schema or application logic will be detected)

### How It Works

1. A GitHub Action defined in `.github/workflows/run-tests.yml` runs the following script:

```bash
./run.sh tests
```

2. This script:
   - Loads environment variables
   - Runs `go test ./...` on all modules
   - Fails the pipeline if any test fails

3. If the pipeline fails:
   - ✅ The pull request **cannot be merged**
   - ✅ You'll get feedback in the **Checks** tab

### Why It Matters

- 🧪 Ensures all changes are safe and tested
- 🔁 Helps identify issues early in the development cycle
- 🔐 Gives you peace of mind when modifying **database migrations**, **models**, or **business logic**
