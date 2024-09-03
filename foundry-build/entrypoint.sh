#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

# Function to print a separator
print_separator() {
    echo "========================================"
}

# Function to print a log message with a timestamp
log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $1"
}

# Install the external modules
if [ -n "$EXTERNAL_MODULES" ]; then
    print_separator
    log "Installing external modules: $EXTERNAL_MODULES"
    for module in $EXTERNAL_MODULES; do
        log "Installing $module..."
        log "forge install --no-commit --no-git $module"
        forge install --no-commit --no-git "$module"
    done
    print_separator
fi

# Parse INIT_PARAMS JSON
PARAMS=$(echo $INIT_PARAMS | jq -r 'join(" ")')
log "Init Params: $PARAMS"
print_separator

# Build the contract
log "Building the contract..."
forge build
print_separator

# Check for test files and run tests if any exist
if ls test/*.sol 1> /dev/null 2>&1; then
    log "Running tests..."
    forge test
else
    log "No tests found, skipping..."
fi
print_separator

# Determine the contract name dynamically
CONTRACT_FILE="src/${CONTRACT_NAME}.sol"

log "Deploying the contract $CONTRACT_NAME..."
print_separator

# Check if RPC_URL ends with a "/" and add it if not
if [[ "${RPC_URL}" != */ ]]; then
    RPC_URL="${RPC_URL}/"
fi

# Combine the RPC URL and RPC Key
FULL_RPC_URL="${RPC_URL}${RPC_KEY}"

# Deploy the contract
if [ -n "$PARAMS" ]; then
    log "forge create $CONTRACT_FILE:$CONTRACT_NAME --rpc-url $FULL_RPC_URL --private-key $WALLET_PRV_KEY --constructor-args $PARAMS"
    forge create "$CONTRACT_FILE:$CONTRACT_NAME" --rpc-url "$FULL_RPC_URL" --private-key "$WALLET_PRV_KEY" --constructor-args $PARAMS
else
    log "forge create $CONTRACT_FILE:$CONTRACT_NAME --rpc-url $FULL_RPC_URL --private-key $WALLET_PRV_KEY"
    forge create "$CONTRACT_FILE:$CONTRACT_NAME" --rpc-url "$FULL_RPC_URL" --private-key "$WALLET_PRV_KEY"
fi

print_separator
log "Deployment completed."
print_separator
