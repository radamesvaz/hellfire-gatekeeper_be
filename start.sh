#!/bin/bash

echo "🚀 Starting Hellfire Gatekeeper API..."

# Run database migrations (don't fail if already applied)
echo "🔄 Running database migrations..."
if ! ./migrate; then
    echo "⚠️  Migration script failed, but continuing with API startup..."
    echo "   This might be normal if migrations are already applied."
fi

# Start the API server
echo "🚀 Starting API server..."
./api
