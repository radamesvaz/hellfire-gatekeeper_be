#!/bin/bash

set -e  # Stop on error

# Colors
GREEN="\033[0;32m"
RED="\033[0;31m"
YELLOW="\033[1;33m"
NC="\033[0m" # No Color

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

  if go test -timeout 2m -v ./tests; then
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
  unit && integration
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
  *)
    echo -e "${RED}⚠️  Command not recognized: $1${NC}"
    echo -e "${YELLOW}👉 Available commands: unit, integration, tests${NC}"
    ;;
esac