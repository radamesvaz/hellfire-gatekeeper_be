#!/bin/bash

set -e  # Stop on error

# Colors
GREEN="\033[0;32m"
RED="\033[0;31m"
YELLOW="\033[1;33m"
NC="\033[0m" # No Color

# Load environment variables from .env file if it exists.
# Prefer a line-based export so values with special characters (e.g. Brevo API keys)
# are not mangled by `xargs` / word-splitting.
if [ -f .env ]; then
  echo -e "${YELLOW}📄 Loading environment variables from .env file...${NC}"
  while IFS= read -r line || [ -n "$line" ]; do
    # Normalize Windows CRLF endings
    line="${line%$'\r'}"
    # Skip comments and blank lines
    case "$line" in
      ''|\#*) continue ;;
    esac
    # Trim leading whitespace
    line="${line#"${line%%[![:space:]]*}"}"
    case "$line" in
      ''|\#*) continue ;;
    esac
    key="${line%%=*}"
    value="${line#*=}"
    value="${value%$'\r'}"
    # Strip optional surrounding quotes
    if [[ "$value" == \"*\" && "$value" == *\" ]]; then
      value="${value:1:${#value}-2}"
    elif [[ "$value" == \'*\' && "$value" == *\' ]]; then
      value="${value:1:${#value}-2}"
    fi
    export "$key=$value"
  done < .env
else
  echo -e "${YELLOW}⚠️  No .env file found. Make sure environment variables are set.${NC}"
fi

# DB helpers (shared by seed commands)
configure_pg_env() {
  export PGHOST="${DB_HOST:-localhost}"
  export PGPORT="${DB_PORT:-5432}"
  export PGUSER="${DB_USER:-${POSTGRES_USER:-}}"
  export PGPASSWORD="${DB_PASSWORD:-${POSTGRES_PASSWORD:-}}"
  export PGDATABASE="${DB_NAME:-${POSTGRES_DB:-}}"

  if [ -z "$PGUSER" ] || [ -z "$PGDATABASE" ]; then
    echo -e "${RED}❌ Set DB_USER and DB_NAME (or POSTGRES_USER and POSTGRES_DB) in .env${NC}"
    exit 1
  fi
}

apply_sql_file() {
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
  # Local default: debug so repository/handler Debug() logs are visible.
  # Override with LOG_LEVEL in .env or the environment (e.g. LOG_LEVEL=info).
  export LOG_LEVEL="${LOG_LEVEL:-debug}"

  echo -e "${YELLOW}🚀 Starting application...${NC}"
  echo -e "${YELLOW}📋 LOG_LEVEL=${LOG_LEVEL}${NC}"
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

  configure_pg_env

  for f in "${sql_files[@]}"; do
    if [[ "$f" == "seeds/002_disable_pagination_mock.sql" ]]; then
      continue
    fi
    echo -e "${YELLOW}   → $f${NC}"
    apply_sql_file "$f"
  done

  echo -e "${GREEN}✅ Seeds applied.${NC}"
}

# Enable extra pagination demo data (for frontend/client testing)
seed_demo_on() {
  echo -e "${YELLOW}🌱 Enabling pagination demo data...${NC}"
  configure_pg_env
  apply_sql_file "seeds/001_multi_tenant_mock.sql"
  echo -e "${GREEN}✅ Pagination demo data enabled.${NC}"
}

# Disable/remove extra pagination demo data (keeps base schema intact)
seed_demo_off() {
  echo -e "${YELLOW}🧹 Disabling pagination demo data...${NC}"
  configure_pg_env
  apply_sql_file "seeds/002_disable_pagination_mock.sql"
  echo -e "${GREEN}✅ Pagination demo data disabled.${NC}"
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

# Local architecture review via Ollama (default model: qwen3:8b).
# Requires Ollama running locally (http://localhost:11434).
#
# What each mode reviews (and whether Ollama is called):
#
#   ./run.sh review
#     Diff: working tree vs HEAD + untracked (new files not yet git-added).
#     Calls Ollama and saves under .ai/reviews/.
#     Use when: "review what I have dirty on disk right now".
#
#   ./run.sh review -staged
#     Diff: only the git index (git diff --staged).
#     Ignores unstaged and untracked changes. Saves under .ai/reviews/.
#     Use when: "review what I'm about to commit".
#
#   ./run.sh review -base master
#     Diff: master...HEAD (commits on your branch vs the default branch).
#     -base main also works: it resolves to master when main does not exist.
#     Ignores a dirty working tree; looks at branch history. Saves under .ai/reviews/.
#     Use when: "review the PR / whole feature vs master".
#     Mutually exclusive with -staged.
#
#   ./run.sh review -dry-run
#     Does NOT call Ollama. Prints the assembled prompt (guidelines + diff).
#     Does NOT save a review file.
#     Use when: debugging prompt size/content without spending model time.
#     Combinable, e.g. ./run.sh review -base master -dry-run
#
# Other useful flags (forwarded to cmd/ai):
#   -model NAME   override model (default qwen3:8b, or OLLAMA_MODEL)
#   -host URL     Ollama base URL (default http://localhost:11434)
#   -save         write review under .ai/reviews/ (already default via run.sh)
#
# Via ./run.sh, reviews are saved under .ai/reviews/ by default for all modes
# except -dry-run. Pass flags as usual; -save is added automatically when missing.
review() {
  echo -e "${YELLOW}🔍 Running local architecture review (Ollama)...${NC}"

  args=("$@")
  wants_save=true
  for arg in "${args[@]}"; do
    case "$arg" in
      -dry-run)
        wants_save=false
        break
        ;;
      -save)
        wants_save=false
        break
        ;;
    esac
  done
  if [ "$wants_save" = true ]; then
    args+=(-save)
  fi

  go run ./cmd/ai review "${args[@]}"
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
  seed-demo-on)
    seed_demo_on
    ;;
  seed-demo-off)
    seed_demo_off
    ;;
  reset)
    reset
    ;;
  review)
    shift
    review "$@"
    ;;
  *)
    echo -e "${RED}⚠️  Command not recognized: $1${NC}"
    echo -e "${YELLOW}👉 Available commands:${NC}"
    echo -e "${YELLOW}   dev        - Start development environment (PostgreSQL + migrations)${NC}"
    echo -e "${YELLOW}   app        - Start the application${NC}"
    echo -e "${YELLOW}   migrate    - Run migrations only${NC}"
    echo -e "${YELLOW}   seed       - Apply seeds/*.sql to local DB (dev only)${NC}"
    echo -e "${YELLOW}   seed-demo-on  - Enable pagination/order demo seed data${NC}"
    echo -e "${YELLOW}   seed-demo-off - Disable pagination/order demo seed data${NC}"
    echo -e "${YELLOW}   unit       - Run unit tests${NC}"
    echo -e "${YELLOW}   integration - Run integration tests${NC}"
    echo -e "${YELLOW}   tests      - Run all tests${NC}"
    echo -e "${YELLOW}   reset      - Reset project (full rebuild)${NC}"
    echo -e "${YELLOW}   review     - Local architecture review with Ollama (saves under .ai/reviews/)${NC}"
    ;;
esac