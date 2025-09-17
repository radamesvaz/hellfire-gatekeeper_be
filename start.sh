#!/bin/bash

set -e

echo "🚀 Starting Hellfire Gatekeeper API..."

# Run database migrations
echo "🔄 Running database migrations..."
./migrate

# Start the API server
echo "🚀 Starting API server..."
./api
