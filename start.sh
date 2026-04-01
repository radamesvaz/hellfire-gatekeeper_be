#!/bin/bash
set -e

echo "🚀 Starting Hellfire Gatekeeper API..."

echo "🔄 Running database migrations..."
./migrate

echo "🚀 Starting API server..."
exec ./api
