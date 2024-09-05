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

# Parse INIT_PARAMS JSON if it is not empty or null
if [ -n "$INIT_PARAMS" ]; then
    PARAMS=$(echo $INIT_PARAMS | jq -r 'join(" ")')
else
    PARAMS=""
fi
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

# Deploy the contract and capture the deployed address
DEPLOY_OUTPUT_FILE=$(mktemp)
if [ -n "$PARAMS" ]; then
    log "forge create $CONTRACT_FILE:$CONTRACT_NAME --rpc-url $FULL_RPC_URL --private-key ************ --constructor-args $PARAMS"
    forge create "$CONTRACT_FILE:$CONTRACT_NAME" --rpc-url "$FULL_RPC_URL" --private-key "$WALLET_PRV_KEY" --constructor-args $PARAMS | tee "$DEPLOY_OUTPUT_FILE"
else
    log "forge create $CONTRACT_FILE:$CONTRACT_NAME --rpc-url $FULL_RPC_URL --private-key ************"
    forge create "$CONTRACT_FILE:$CONTRACT_NAME" --rpc-url "$FULL_RPC_URL" --private-key "$WALLET_PRV_KEY" | tee "$DEPLOY_OUTPUT_FILE"
fi

# Extract the deployed contract address from the output
CONTRACT_ADDRESS=$(grep -oP 'Deployed to: \K(0x[a-fA-F0-9]{40})' "$DEPLOY_OUTPUT_FILE")

print_separator
log "Deployment completed. Contract Address: $CONTRACT_ADDRESS"
print_separator

# Verify the contract if BlockExplorer details are provided
if [ -n "$ETHERSCAN_API_KEY" ]; then
    log "Verifying the contract on BlockExplorer..."
    if [ -n "$PARAMS" ]; then
        echo "$PARAMS" > ./params.txt
        log "forge verify-contract $CONTRACT_ADDRESS $CONTRACT_NAME --chain-id $CHAIN_ID --constructor-args-path ./params.txt"
        forge verify-contract $CONTRACT_ADDRESS $CONTRACT_NAME --chain-id $CHAIN_ID --constructor-args-path ./params.txt
    else
        log "forge verify-contract $CONTRACT_ADDRESS $CONTRACT_NAME --chain-id $CHAIN_ID"
        forge verify-contract $CONTRACT_ADDRESS $CONTRACT_NAME --chain-id $CHAIN_ID
    fi
    print_separator
    log "Contract verification completed."
    print_separator
else
    log "BlockExplorer details not provided, skipping verification."
fi