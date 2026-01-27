# 📋 Instrucciones para Replicar el Sistema de Migraciones

Este documento contiene las instrucciones completas para replicar el sistema de migraciones de base de datos en un nuevo servicio Go.

## 🎯 Objetivo

Configurar un sistema de migraciones de base de datos usando `golang-migrate` que permita:
- Aplicar migraciones de forma segura y versionada
- Revertir migraciones cuando sea necesario
- Ejecutar migraciones en diferentes entornos (local, Docker, producción)
- Manejar errores y estados inconsistentes de la base de datos

## 📦 Dependencias Requeridas

### 1. Dependencias de Go

Agregar al archivo `go.mod`:

```go
require (
    github.com/golang-migrate/migrate/v4 v4.19.0
    github.com/joho/godotenv v1.5.1
    github.com/lib/pq v1.10.9
    github.com/rs/zerolog v1.34.0  // o tu logger preferido
)
```

### 2. Instalación de golang-migrate CLI (Opcional, para crear migraciones)

**Windows:**
```powershell
choco install migrate
```

**Linux/Mac:**
```bash
brew install migrate
# o
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

## 🗂️ Estructura de Directorios

Crear la siguiente estructura:

```
tu-proyecto/
├── cmd/
│   └── migrate/
│       └── main.go
├── migrations/
│   ├── README.md
│   ├── 000001_init_schema.up.sql
│   └── 000001_init_schema.down.sql
├── docker-compose.yml
├── go.mod
└── .env
```

## 🔧 Configuración Paso a Paso

### Paso 1: Crear el Comando de Migración

Crear `cmd/migrate/main.go`:

```go
package main

import (
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	// Importa tu logger aquí
)

func main() {
	// Cargar variables de entorno desde .env (opcional)
	err := godotenv.Load()
	if err != nil {
		// .env es opcional, especialmente en producción
	}

	// Configurar logger (ajusta según tu implementación)
	// logger.Init(os.Getenv("LOG_LEVEL"))

	// Obtener configuración de base de datos desde variables de entorno
	databaseURL := os.Getenv("DATABASE_URL")
	dbUser := firstNonEmpty(
		os.Getenv("POSTGRES_USER"),
		os.Getenv("PGUSER"),
		os.Getenv("DB_USER"),
	)
	dbPassword := firstNonEmpty(
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("PGPASSWORD"),
		os.Getenv("DB_PASSWORD"),
	)
	dbHost := firstNonEmpty(
		os.Getenv("DB_HOST"),
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("PGHOST"),
		"localhost",
	)
	dbPort := firstNonEmpty(
		os.Getenv("DB_PORT"),
		os.Getenv("POSTGRES_PORT"),
		os.Getenv("PGPORT"),
		"5432",
	)
	dbName := firstNonEmpty(
		os.Getenv("POSTGRES_DB"),
		os.Getenv("PGDATABASE"),
		os.Getenv("DB_NAME"),
	)

	// Validar que tenemos las variables necesarias
	if databaseURL == "" && (dbUser == "" || dbPassword == "" || dbHost == "" || dbPort == "" || dbName == "") {
		fmt.Fprintf(os.Stderr, "Error: Missing required database environment variables\n")
		os.Exit(1)
	}

	// Construir DSN (Data Source Name)
	var dsn string
	if databaseURL != "" {
		dsn = databaseURL
	} else {
		sslMode := "require"
		lowerHost := strings.ToLower(dbHost)
		if lowerHost == "localhost" || lowerHost == "127.0.0.1" || lowerHost == "::1" {
			sslMode = "disable"
		}
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			dbHost, dbPort, dbUser, dbPassword, dbName, sslMode)
	}

	// Conectar a la base de datos
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Could not connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Probar la conexión
	if err := db.Ping(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Could not ping database: %v\n", err)
		os.Exit(1)
	}

	// Crear instancia del driver de migrate
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Could not create migrate driver: %v\n", err)
		os.Exit(1)
	}

	// Crear instancia de migrate
	m, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Could not create migrate instance: %v\n", err)
		os.Exit(1)
	}

	// Ejecutar migraciones
	fmt.Println("Running database migrations...")
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		// Manejar base de datos en estado "dirty"
		if strings.Contains(err.Error(), "Dirty database version") {
			re := regexp.MustCompile(`Dirty database version (\d+)`)
			matches := re.FindStringSubmatch(err.Error())
			if len(matches) > 1 {
				version := matches[1]
				fmt.Printf("Warning: Database is dirty at version %s, cleaning and resetting...\n", version)
				
				// Forzar versión a 1 (primera migración)
				if forceErr := m.Force(1); forceErr != nil {
					fmt.Fprintf(os.Stderr, "Error: Could not force version to 1: %v\n", forceErr)
					os.Exit(1)
				}
				
				// Limpiar tablas (ajusta según tus tablas)
				fmt.Println("Dropping all tables to start fresh...")
				// Agrega aquí las sentencias DROP para tus tablas
				
				// Reintentar migraciones
				fmt.Println("Running migrations from scratch...")
				if retryErr := m.Up(); retryErr != nil && retryErr != migrate.ErrNoChange {
					fmt.Fprintf(os.Stderr, "Error: Could not run migrations after reset: %v\n", retryErr)
					os.Exit(1)
				}
			}
		} else {
			fmt.Fprintf(os.Stderr, "Error: Could not run migrations: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Println("Database migrations completed successfully!")
}

// firstNonEmpty retorna el primer string no vacío de la lista
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
```

### Paso 2: Crear Directorio de Migraciones

```bash
mkdir migrations
```

### Paso 3: Crear Primera Migración

**Ejemplo: `migrations/000001_init_schema.up.sql`**

```sql
-- Crear tipos ENUM si no existen
DO $$ BEGIN
    CREATE TYPE order_status AS ENUM ('pending', 'preparing', 'ready', 'delivered', 'cancelled');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- Crear tabla de usuarios
CREATE TABLE users (
    id_user SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    phone VARCHAR(20),
    password_hash VARCHAR(255),
    created_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Crear tabla de productos
CREATE TABLE products (
    id_product SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(10,2) NOT NULL,
    available BOOLEAN DEFAULT TRUE,
    created_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Crear tabla de órdenes
CREATE TABLE orders (
    id_order SERIAL PRIMARY KEY,
    id_user INT NOT NULL,
    status order_status DEFAULT 'pending',
    total_price DECIMAL(10,2) NOT NULL,
    created_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (id_user) REFERENCES users(id_user) ON DELETE CASCADE
);
```

**Ejemplo: `migrations/000001_init_schema.down.sql`**

```sql
-- Revertir en orden inverso (primero dependientes, luego principales)
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS users;
DROP TYPE IF EXISTS order_status;
```

### Paso 4: Configurar Docker Compose

Crear o actualizar `docker-compose.yml` con la siguiente configuración completa:

```yaml
version: "2.4"

services:
  # Servicio de PostgreSQL
  postgres_db:
    image: postgres:15
    container_name: tu_servicio_postgres
    restart: always
    # Healthcheck para verificar que PostgreSQL esté listo antes de ejecutar migraciones
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER} -d ${POSTGRES_DB}"]
      interval: 10s
      timeout: 5s
      retries: 5
    environment:
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
      POSTGRES_DB: ${POSTGRES_DB}
    ports:
      - "5432:5432"  # Expone PostgreSQL en el puerto 5432 del host
    volumes:
      # Volumen persistente para los datos de PostgreSQL
      - postgres_data:/var/lib/postgresql/data

  # Servicio de migraciones
  migrate:
    image: migrate/migrate
    container_name: tu_servicio_migrations
    # Espera a que PostgreSQL esté saludable antes de ejecutar
    depends_on:
      postgres_db:
        condition: service_healthy
    volumes:
      # Monta el directorio de migraciones local en el contenedor
      - ./migrations:/migrations
    command: [ 
      "-path", "/migrations", 
      "-database", 
      "postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres_db:5432/${POSTGRES_DB}?sslmode=disable", 
      "up" 
    ]

  # Servicio del backend (aplicación Go)
  backend:
    build: .  # Construye la imagen desde el Dockerfile en la raíz
    container_name: tu_servicio_backend
    # Espera a que PostgreSQL y las migraciones estén listas
    depends_on:
      - postgres_db
      - migrate
    ports:
      - "8080:8080"  # Expone la API en el puerto 8080
    environment:
      # Variables de entorno para la conexión a la base de datos
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
      POSTGRES_DB: ${POSTGRES_DB}
      DB_HOST: ${DB_HOST:-postgres_db}  # Usa 'postgres_db' por defecto en Docker
      DB_PORT: ${DB_PORT:-5432}
      # Agrega aquí otras variables de entorno que necesite tu aplicación
      # JWT_SECRET: ${JWT_SECRET}
      # JWT_EXPIRATION_MINUTES: ${JWT_EXPIRATION_MINUTES}
      # LOG_LEVEL: ${LOG_LEVEL:-info}
    volumes:
      # Monta directorios locales si necesitas persistencia de archivos
      # - ./uploads:/app/uploads
      # - ./logs:/app/logs

# Volúmenes nombrados para persistencia de datos
volumes:
  postgres_data:
    # Los datos de PostgreSQL se mantienen incluso si el contenedor se elimina
```

#### Explicación de cada servicio:

**1. `postgres_db` (Base de datos PostgreSQL)**
- Usa la imagen oficial de PostgreSQL versión 15
- `healthcheck`: Verifica que PostgreSQL esté listo antes de que otros servicios dependan de él
- `restart: always`: Reinicia automáticamente si el contenedor se detiene
- `volumes`: Persiste los datos en un volumen nombrado para que no se pierdan al eliminar el contenedor

**2. `migrate` (Ejecutor de migraciones)**
- Usa la imagen oficial de `golang-migrate`
- `depends_on` con `condition: service_healthy`: Espera a que PostgreSQL esté completamente listo
- Monta el directorio `./migrations` para acceder a los archivos SQL
- Ejecuta automáticamente `migrate up` al iniciar

**3. `backend` (Aplicación Go)**
- Construye desde un `Dockerfile` en la raíz del proyecto
- `depends_on`: Garantiza que PostgreSQL y las migraciones se ejecuten primero
- Expone el puerto 8080 para acceder a la API
- Recibe todas las variables de entorno necesarias desde el archivo `.env`

#### Comandos útiles de Docker Compose:

```bash
# Iniciar todos los servicios
docker-compose up -d

# Iniciar solo PostgreSQL (sin migraciones ni backend)
docker-compose up postgres_db -d

# Ejecutar solo las migraciones
docker-compose up migrate

# Ver logs de un servicio específico
docker-compose logs -f postgres_db
docker-compose logs -f migrate
docker-compose logs -f backend

# Detener todos los servicios
docker-compose down

# Detener y eliminar volúmenes (⚠️ elimina los datos de la BD)
docker-compose down -v

# Reconstruir y reiniciar servicios
docker-compose up --build -d

# Ver el estado de los servicios
docker-compose ps
```

#### Notas importantes:

1. **Variables de entorno**: Todas las variables `${VARIABLE}` deben estar definidas en tu archivo `.env` o en el sistema
2. **Healthcheck**: El healthcheck de PostgreSQL es crucial para que las migraciones esperen correctamente
3. **Volúmenes**: El volumen `postgres_data` persiste los datos. Si necesitas empezar desde cero, usa `docker-compose down -v`
4. **Red interna**: Los servicios se comunican usando los nombres de servicio como hostnames (ej: `postgres_db` en lugar de `localhost`)
5. **Puertos**: Asegúrate de que los puertos 5432 y 8080 no estén en uso en tu máquina local

### Paso 5: Crear Archivo .env

Crear `.env` en la raíz del proyecto:

```env
# Base de datos
POSTGRES_USER=tu_usuario
POSTGRES_PASSWORD=tu_password_seguro
POSTGRES_DB=tu_base_de_datos
DB_HOST=localhost
DB_PORT=5432

# Opcional: URL completa de base de datos
# DATABASE_URL=postgres://usuario:password@localhost:5432/nombre_db?sslmode=disable
```

### Paso 6: Crear Script de Ejecución (Opcional)

Crear `run.sh` (o `run.ps1` para Windows):

**run.sh:**
```bash
#!/bin/bash

set -e

# Cargar variables de entorno
if [ -f .env ]; then
  export $(cat .env | grep -v '^#' | xargs)
fi

# Comando para ejecutar migraciones
migrate() {
  echo "🔄 Running migrations..."
  go run cmd/migrate/main.go
  echo "✅ Migrations completed!"
}

# Comando para iniciar desarrollo
dev() {
  echo "🚀 Starting development environment..."
  docker-compose up postgres_db -d
  sleep 5
  go run cmd/migrate/main.go
  echo "✅ Development environment ready!"
}

case "$1" in
  migrate)
    migrate
    ;;
  dev)
    dev
    ;;
  *)
    echo "Usage: $0 {migrate|dev}"
    exit 1
    ;;
esac
```

**run.ps1 (Windows PowerShell):**
```powershell
param(
    [Parameter(Mandatory=$true)]
    [ValidateSet("migrate", "dev")]
    [string]$Command
)

# Cargar variables de entorno desde .env
if (Test-Path .env) {
    Get-Content .env | ForEach-Object {
        if ($_ -match '^([^#][^=]+)=(.*)$') {
            [Environment]::SetEnvironmentVariable($matches[1], $matches[2], "Process")
        }
    }
}

switch ($Command) {
    "migrate" {
        Write-Host "🔄 Running migrations..." -ForegroundColor Yellow
        go run cmd/migrate/main.go
        Write-Host "✅ Migrations completed!" -ForegroundColor Green
    }
    "dev" {
        Write-Host "🚀 Starting development environment..." -ForegroundColor Yellow
        docker-compose up postgres_db -d
        Start-Sleep -Seconds 5
        go run cmd/migrate/main.go
        Write-Host "✅ Development environment ready!" -ForegroundColor Green
    }
}
```

## 🚀 Uso del Sistema de Migraciones

### Crear una Nueva Migración

```bash
migrate create -ext sql -dir migrations -seq nombre_descriptivo_de_la_migracion
```

Esto creará dos archivos:
- `migrations/XXXXXX_nombre_descriptivo_de_la_migracion.up.sql`
- `migrations/XXXXXX_nombre_descriptivo_de_la_migracion.down.sql`

### Aplicar Migraciones

**Opción 1: Usando el comando Go**
```bash
go run cmd/migrate/main.go
```

**Opción 2: Usando Docker Compose**
```bash
docker-compose up migrate
```

**Opción 3: Usando el script**
```bash
./run.sh migrate
# o en Windows PowerShell:
.\run.ps1 migrate
```

### Revertir Migraciones

Para revertir la última migración, necesitarías modificar `cmd/migrate/main.go` para aceptar argumentos:

```go
// Al inicio de main()
if len(os.Args) > 1 {
    command := os.Args[1]
    if command == "down" {
        if err := m.Down(); err != nil && err != migrate.ErrNoChange {
            fmt.Fprintf(os.Stderr, "Error: Could not rollback migrations: %v\n", err)
            os.Exit(1)
        }
        fmt.Println("Migrations rolled back successfully!")
        return
    }
}
```

Luego ejecutar:
```bash
go run cmd/migrate/main.go down
```

## 📝 Mejores Prácticas

1. **Nombres descriptivos**: Usa nombres claros para las migraciones (ej: `add_user_email_index`, `create_orders_table`)

2. **Migraciones atómicas**: Cada migración debe ser una unidad lógica completa

3. **Siempre crear archivos `.down.sql`**: Permite revertir cambios si es necesario

4. **Probar localmente primero**: Ejecuta las migraciones en desarrollo antes de commitear

5. **Orden de dependencias**: En `.down.sql`, elimina primero las tablas dependientes

6. **Manejo de ENUMs**: Usa bloques `DO $$ BEGIN ... EXCEPTION ... END $$` para crear ENUMs de forma segura

7. **Versionado**: Nunca modifiques migraciones ya aplicadas. Crea nuevas migraciones para cambios

## 🔍 Verificación

Para verificar que las migraciones se aplicaron correctamente:

```sql
-- Conectarse a PostgreSQL
psql -U tu_usuario -d tu_base_de_datos

-- Ver el estado de las migraciones
SELECT * FROM schema_migrations;
```

## ⚠️ Troubleshooting

### Error: "Dirty database version"
- Significa que una migración falló a mitad de camino
- El sistema intentará limpiar automáticamente
- Si persiste, puedes forzar manualmente: `m.Force(version)`

### Error: "no migration found for version 0"
- Generalmente ocurre cuando hay inconsistencias
- El sistema intentará limpiar y reiniciar desde cero

### Error: "connection refused"
- Verifica que PostgreSQL esté corriendo
- Verifica las variables de entorno (host, port, user, password)

## 📚 Recursos Adicionales

- [Documentación oficial de golang-migrate](https://github.com/golang-migrate/migrate)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)

---

**Nota**: Ajusta los nombres de tablas, tipos ENUM y estructura según las necesidades de tu nuevo servicio.

