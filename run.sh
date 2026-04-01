#!/bin/bash

set -e  # Stop on error

# Colors
GREEN="\033[0;32m"
RED="\033[0;31m"
YELLOW="\033[1;33m"
NC="\033[0m" # No Color

# Load environment variables from .env file if it exists
if [ -f .env ]; then
  echo -e "${YELLOW}📄 Loading environment variables from .env file...${NC}"
  export $(cat .env | grep -v '^#' | xargs)
else
  echo -e "${YELLOW}⚠️  No .env file found. Make sure environment variables are set.${NC}"
fi

# Run all unit tests
unit() {
  echo -e "${YELLOW}🧪 Running unit tests...${NC}"
  go clean -testcache
    
    start=$(date +%s)

  if go test ./internal/...; then
    end=$(date +%s)
    runtime=$((end - start))
    echo -e "${GREEN}✅ All UNIT tests passed in ${runtime}s${NC}"
  else
    echo -e "${RED}❌ UNIT tests failed.${NC}"
    exit 1
  fi
}

# Run all integration tests
integration() {
  echo -e "${YELLOW}🌐 Running integration tests...${NC}"
  go clean -testcache

      start=$(date +%s)

  if go test -timeout 15m -v ./tests; then
    end=$(date +%s)
    runtime=$((end - start))
    echo -e "${GREEN}✅ All INTEGRATION tests passed in ${runtime}s${NC}"
  else
    echo -e "${RED}❌ INTEGRATION tests failed.${NC}"
    exit 1
  fi
}

# Run all tests
tests() {
  echo "🧹 Running go mod tidy..."
  go mod tidy

  unit && integration
}

# Start development environment
dev() {
  echo -e "${YELLOW}🚀 Starting development environment...${NC}"
  
  echo -e "${YELLOW}📦 Starting PostgreSQL with Docker...${NC}"
  docker-compose up postgres_db -d
  
  echo -e "${YELLOW}⏳ Waiting for PostgreSQL to be ready...${NC}"
  sleep 5
  
  echo -e "${YELLOW}🔄 Running migrations...${NC}"
  go run ./cmd/migrate

  echo -e "${GREEN}✅ Development environment ready!${NC}"
  echo -e "${YELLOW}👉 Run './run.sh app' to start the application${NC}"
  echo -e "${YELLOW}👉 Optional: './run.sh seed' for local multi-tenant demo data${NC}"
}

# Start the application
app() {
  echo -e "${YELLOW}🚀 Starting application...${NC}"
  echo -e "${YELLOW}🌐 Server will be available at http://localhost:8080${NC}"
  go run cmd/api/main.go
}

# Run migrations only
migrate() {
  echo -e "${YELLOW}🔄 Running migrations...${NC}"
  go run ./cmd/migrate
  echo -e "${GREEN}✅ Migrations completed!${NC}"
}

# Apply local SQL seeds (never used on Render). Requires psql or Docker postgres_db.
seed() {
  echo -e "${YELLOW}🌱 Applying local dev seeds from seeds/*.sql ...${NC}"

  shopt -s nullglob
  local sql_files=(seeds/*.sql)
  shopt -u nullglob

  if [ ${#sql_files[@]} -eq 0 ]; then
    echo -e "${RED}❌ No .sql files found in seeds/${NC}"
    exit 1
  fi

  export PGHOST="${DB_HOST:-localhost}"
  export PGPORT="${DB_PORT:-5432}"
  export PGUSER="${DB_USER:-${POSTGRES_USER:-}}"
  export PGPASSWORD="${DB_PASSWORD:-${POSTGRES_PASSWORD:-}}"
  export PGDATABASE="${DB_NAME:-${POSTGRES_DB:-}}"

  if [ -z "$PGUSER" ] || [ -z "$PGDATABASE" ]; then
    echo -e "${RED}❌ Set DB_USER and DB_NAME (or POSTGRES_USER and POSTGRES_DB) in .env${NC}"
    exit 1
  fi

  apply_sql() {
    local f="$1"
    if command -v psql >/dev/null 2>&1; then
      psql -v ON_ERROR_STOP=1 -f "$f"
    else
      if [ -z "${POSTGRES_USER:-}" ] || [ -z "${POSTGRES_DB:-}" ]; then
        echo -e "${RED}❌ psql not found; for Docker fallback set POSTGRES_USER and POSTGRES_DB in .env${NC}"
        exit 1
      fi
      echo -e "${YELLOW}   (using docker-compose exec postgres_db — install psql for direct use)${NC}"
      docker-compose exec -T postgres_db psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -v ON_ERROR_STOP=1 < "$f"
    fi
  }

  for f in "${sql_files[@]}"; do
    echo -e "${YELLOW}   → $f${NC}"
    apply_sql "$f"
  done

  echo -e "${GREEN}✅ Seeds applied.${NC}"
}

# Reset project: full rebuild
reset() {
  echo "🚧 Stopping containers..."
  docker-compose down -v

  echo "🧹 Cleaning old build artifacts..."
  docker system prune -f

  echo "🔧 Rebuilding and starting services..."
  docker-compose up --build -d

  echo "✅ Project is up and running!"
}

case "$1" in
  unit)
    unit
    ;;
  integration)
    integration
    ;;
  tests)
    tests
    ;;
  dev)
    dev
    ;;
  app)
    app
    ;;
  migrate)
    migrate
    ;;
  seed)
    seed
    ;;
  reset)
    reset
    ;;
  *)
    echo -e "${RED}⚠️  Command not recognized: $1${NC}"
    echo -e "${YELLOW}👉 Available commands:${NC}"
    echo -e "${YELLOW}   dev        - Start development environment (PostgreSQL + migrations)${NC}"
    echo -e "${YELLOW}   app        - Start the application${NC}"
    echo -e "${YELLOW}   migrate    - Run migrations only${NC}"
    echo -e "${YELLOW}   seed       - Apply seeds/*.sql to local DB (dev only)${NC}"
    echo -e "${YELLOW}   unit       - Run unit tests${NC}"
    echo -e "${YELLOW}   integration - Run integration tests${NC}"
    echo -e "${YELLOW}   tests      - Run all tests${NC}"
    echo -e "${YELLOW}   reset      - Reset project (full rebuild)${NC}"
    ;;
esac