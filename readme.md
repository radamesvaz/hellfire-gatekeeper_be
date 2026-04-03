# 🚀 Hellfire Gatekeeper Backend

A robust Go-based backend API for a bakery management system with PostgreSQL database integration, featuring product management, order processing, authentication, and image handling.

## 🚀 Quick Start

### Prerequisites

- Go 1.19+
- Docker and Docker Compose
- PostgreSQL (via Docker)

### Setup

1. **Clone the repository:**
```bash
git clone <repository-url>
cd hellfire-gatekeeper_be
```

2. **Create a `.env` file in the project root:**
```env
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres123
POSTGRES_DB=bakery_db
DB_HOST=localhost
DB_PORT=5432
JWT_SECRET=your_very_secure_jwt_secret_here
JWT_EXPIRATION_MINUTES=60
PORT=8080
```

3. **Start the development environment:**
```bash
./run.sh dev
```

4. **Run the application:**
```bash
./run.sh app
```

## 📋 Development Scripts

### Main Scripts

- **`./run.sh dev`** - Start PostgreSQL and run migrations
- **`./run.sh app`** - Run the application
- **`./run.sh migrate`** - Run migrations only
- **`./run.sh tests`** - Run all tests
- **`./run.sh unit`** - Run unit tests
- **`./run.sh integration`** - Run integration tests
- **`./run.sh reset`** - Reset the complete project

### Manual Commands

If you prefer not to use the scripts:

```bash
# 1. Start PostgreSQL
docker-compose up postgres_db -d

# 2. Run migrations
go run cmd/migrate/main.go up

# 3. Run application
go run cmd/api/main.go
```

## 🔧 Useful Commands

### Check PostgreSQL
```bash
docker ps
```

### Stop PostgreSQL
```bash
docker-compose down
```

### View PostgreSQL logs
```bash
docker-compose logs postgres_db
```

## 📚 API Endpoints

### OpenAPI (pagination & list search)

Machine-readable contract for **cursor-paginated** list endpoints is in **`docs/openapi.yaml`**: `GET /products`, `GET /t/{tenant_slug}/products`, and `GET /auth/orders` (query params `limit`, `cursor`, product search `q`, order filter `id_user`, envelope `{ "items", "next_cursor" }`, separate cursor formats for products vs orders).

**View the spec**

- [Swagger Editor](https://editor.swagger.io) — *File* → import `docs/openapi.yaml`
- IDE extensions (OpenAPI / Swagger preview) on that file
- Local preview: `npx --yes @redocly/cli preview-docs docs/openapi.yaml`

**Lint (optional, Node.js)**

```bash
npx --yes @redocly/cli lint docs/openapi.yaml
```

You can generate client types/SDKs with tools such as **openapi-generator** from the same file.

### Products
- `GET /products` - Get all products
- `GET /products/{id}` - Get product by ID
- `POST /auth/products` - Create product (requires authentication)
- `PUT /auth/products/{id}` - Update product (requires authentication)
- `PATCH /auth/products/{id}` - Update product status (requires authentication)

### Product Images
- `POST /auth/products/{id}/images` - Add product images (requires authentication)
- `PUT /auth/products/{id}/images` - Replace product images (requires authentication)
- `DELETE /auth/products/{id}/images` - Delete product image (requires authentication)

### Orders
- `GET /auth/orders` - Get all orders (requires authentication)
- `GET /auth/orders?ignore_status=true` - Get all orders including deleted ones
- `GET /auth/orders?status=pending` - Filter orders by status
- `GET /auth/orders/{id}` - Get order by ID (requires authentication)
- `POST /orders` - Create order (public endpoint)
- `PATCH /auth/orders/{id}` - Update order (requires authentication)

### Authentication
- `POST /login` - Login
- `POST /register` - Register

### Health Check
- `GET /health` - Health check endpoint

## 🗄️ Database

This project uses **PostgreSQL** with automatic migrations. Migrations are located in the `migrations/` directory.

### Order Statuses
- `pending` - Pending
- `preparing` - In preparation
- `ready` - Ready
- `delivered` - Delivered
- `cancelled` - Cancelled
- `deleted` - Deleted

## 🏗️ Architecture

### Project Structure
```
├── cmd/
│   ├── api/           # Main application entry point
│   └── migrate/        # Database migration runner
├── internal/
│   ├── handlers/       # HTTP request handlers
│   ├── middleware/     # HTTP middleware (auth, CORS)
│   ├── repository/     # Data access layer
│   ├── services/       # Business logic layer
│   └── errors/         # Error handling
├── migrations/         # Database schema migrations
├── model/             # Data models
├── tests/             # Test files
└── uploads/           # File uploads directory
```

### Key Features

- **Authentication**: JWT-based authentication with secure token management
- **Product Management**: Full CRUD operations for bakery products
- **Order Processing**: Complete order lifecycle management
- **Image Handling**: Product image upload and management
- **Database Migrations**: Automated schema management
- **Health Monitoring**: Database connection health checks
- **CORS Support**: Configurable cross-origin resource sharing
- **Connection Pooling**: Optimized database connection management

## 🧪 Testing

### Run Tests
```bash
go test ./...
```

### Tests with Docker
```bash
./run.sh tests
```

### Test Categories
- **Unit Tests**: Individual component testing
- **Integration Tests**: Database and service integration testing
- **Migration Tests**: Database schema validation

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

## 🚀 Deployment

### Environment Variables

The application supports multiple database connection methods:

- **DATABASE_URL**: Full connection string (preferred for production)
- **Discrete variables**: Individual database configuration variables
- **Connection pooling**: Configurable connection pool settings

### Production Considerations

- Database connection pooling is optimized for production workloads
- Health checks ensure service availability
- CORS is configured for production domains
- File uploads are handled securely
- JWT tokens are properly validated and expired

## 🔒 Security Features

- JWT-based authentication
- Password hashing with bcrypt
- CORS protection
- Input validation
- SQL injection prevention
- Secure file upload handling

## 📝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests to ensure everything works
5. Submit a pull request

## 📄 License

This project is licensed under the MIT License.