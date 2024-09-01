#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

# Step 1: Install the external modules
if [ -n "$EXTERNAL_MODULES" ]; then
    echo "Installing external modules: $EXTERNAL_MODULES"
    for module in $EXTERNAL_MODULES; do
        echo "Installing $module..."
        echo "forge install --no-commit --no-git "$module""
        forge install --no-commit --no-git "$module"
    done
fi

# Step 2: Copy local modules from ConfigMap
if [ -n "$LOCAL_MODULES" ]; then
    echo "Copying local modules..."
    for module in $LOCAL_MODULES; do
        cp -r "/config/$module" "/home/foundryuser/expedio-kontract-deployer/src/$module"
    done
fi

# Step 3: Build the contract
echo "Building the contract..."
forge build

# Step 4: Check for test files and run tests if any exist
if ls test/*.sol 1> /dev/null 2>&1; then
    echo "Running tests..."
    forge test
else
    echo "No tests found, skipping..."
fi

# Step 5: Determine the contract name dynamically
CONTRACT_FILE="src/${CONTRACT_NAME}.sol"

echo "Deploying the contract $CONTRACT_NAME..."

# Check if RPC_URL ends with a "/" and add it if not
if [[ "${RPC_URL}" != */ ]]; then
    RPC_URL="${RPC_URL}/"
fi

# Combine the RPC URL and RPC Key
FULL_RPC_URL="${RPC_URL}${RPC_KEY}"

# Step 6: Deploy the contract
forge create --rpc-url "$FULL_RPC_URL" --private-key "$WALLET_PRV_KEY" "$CONTRACT_FILE:$CONTRACT_NAME"

echo "Deployment completed."
