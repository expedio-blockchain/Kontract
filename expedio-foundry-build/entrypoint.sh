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

# Step 3: Deploy the contract
echo "Deploying the contract..."
forge create --rpc-url $RPC_URL --private-key $PRIVATE_KEY src/Contract.sol:Contract

echo "Deployment completed."
