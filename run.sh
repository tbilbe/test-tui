#!/bin/bash

# Seven Test TUI - Quick Start Guide
# This script helps you run the TUI with proper configuration

echo "╔══════════════════════════════════════════════════════════════╗"
echo "║              Seven Test TUI - Quick Start                    ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""

# Check if .env file exists
if [ ! -f .env ]; then
    echo "⚠️  No .env file found. Creating from template..."
    if [ -f .env.example ]; then
        cp .env.example .env
        echo "✓ Created .env file from .env.example"
        echo ""
        echo "📝 Please edit .env and add your configuration:"
        echo "   - API_ENDPOINT"
        echo "   - USER_POOL_ID"
        echo "   - CLIENT_ID"
        echo "   - PREFIX"
        echo ""
        echo "Then run this script again."
        exit 1
    else
        echo "❌ .env.example not found. Please create .env manually."
        exit 1
    fi
fi

# Load environment variables
export $(cat .env | grep -v '^#' | xargs)

# Check required variables
REQUIRED_VARS=("API_ENDPOINT" "USER_POOL_ID" "CLIENT_ID" "PREFIX")
MISSING_VARS=()

for var in "${REQUIRED_VARS[@]}"; do
    if [ -z "${!var}" ]; then
        MISSING_VARS+=("$var")
    fi
done

if [ ${#MISSING_VARS[@]} -gt 0 ]; then
    echo "❌ Missing required environment variables:"
    for var in "${MISSING_VARS[@]}"; do
        echo "   - $var"
    done
    echo ""
    echo "Please update your .env file with these values."
    exit 1
fi

# Check AWS credentials
echo "🔐 Checking AWS credentials..."
if ! aws sts get-caller-identity &>/dev/null; then
    echo "❌ AWS credentials not configured or expired"
    echo ""
    echo "Please run:"
    echo "  aws sso login --profile seven_engineer_seven_dev-339713102567"
    echo "  export AWS_PROFILE=seven_engineer_seven_dev-339713102567"
    exit 1
fi

echo "✓ AWS credentials valid"
echo ""

# Display configuration
echo "📋 Configuration:"
echo "   API Endpoint: $API_ENDPOINT"
echo "   User Pool ID: $USER_POOL_ID"
echo "   Prefix: $PREFIX"
echo "   Region: ${AWS_REGION:-eu-west-2}"
echo ""

# Build if needed
if [ ! -f ./seven-test-tui ]; then
    echo "🔨 Building application..."
    go build -o seven-test-tui cmd/main.go
    echo "✓ Build complete"
    echo ""
fi

# Run the TUI
echo "🚀 Starting Seven Test TUI..."
echo ""

# Export variables and run
export API_ENDPOINT
export USER_POOL_ID
export CLIENT_ID
export PREFIX
export AWS_REGION

./seven-test-tui
