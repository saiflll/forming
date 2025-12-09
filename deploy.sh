#!/bin/bash

# Deployment script untuk Forming App
echo "🚀 Starting Forming App Deployment..."

# 1. Stop existing container
echo "📦 Stopping existing container..."
docker-compose down

# 2. Rebuild image (with updated Dockerfile)
echo "🔨 Building new image..."
docker-compose build --no-cache

# 3. Start container
echo "▶️  Starting container..."
docker-compose up -d

# 4. Wait for container to be ready
echo "⏳ Waiting for container to start..."
sleep 3

# 5. Show logs
echo "📋 Container logs:"
docker logs forming-app --tail 50

echo ""
echo "✅ Deployment complete!"
echo "🌐 Dashboard: http://localhost:3000"
echo ""
echo "📊 To monitor logs:"
echo "   docker logs forming-app -f"
