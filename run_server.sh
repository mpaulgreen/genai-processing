#!/bin/bash

# Function to load environment variables
load_env() {
    if [ -f .env ]; then
        echo "Loading environment from .env..."
        set -a
        source .env
        set +a
        echo "Environment loaded successfully"
    else
        echo "Warning: .env file not found"
    fi
}

# Function to verify critical env vars
verify_env() {
    local missing_vars=()
    
    if [ -z "$OPENAI_API_KEY" ]; then
        missing_vars+=("OPENAI_API_KEY")
    fi
    
    if [ -z "$CLAUDE_API_KEY" ]; then
        missing_vars+=("CLAUDE_API_KEY")
    fi
    
    if [ ${#missing_vars[@]} -ne 0 ]; then
        echo "Error: Missing required environment variables: ${missing_vars[*]}"
        exit 1
    fi
    
    echo "All required environment variables are set"
}

# Main execution
load_env
verify_env

echo "Starting server..."
exec ./server
