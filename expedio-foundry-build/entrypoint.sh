#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

# Step 1: Build the contract
echo "Building the contract..."
forge build

# Step 2: Check for test files and run tests if any exist
if ls test/*.sol 1> /dev/null 2>&1; then
    echo "Running tests..."
    forge test
else
    echo "No tests found, skipping..."
fi

# Step 3: Determine the contract name dynamically
CONTRACT_FILE="src/Contract.sol"
CONTRACT_NAME=$(grep -oP 'contract \K\w+' "$CONTRACT_FILE")

if [ -z "$CONTRACT_NAME" ]; then
    echo "Error: No contract name found in $CONTRACT_FILE"
    exit 1
fi

echo "Deploying the contract $CONTRACT_NAME..."

# Step 4: Deploy the contract
forge create --rpc-url $RPC_URL --private-key $PRIVATE_KEY "$CONTRACT_FILE:$CONTRACT_NAME"

echo "Deployment completed."
