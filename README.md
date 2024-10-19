# Kontract Operator

Kontract is a Kubernetes operator designed to simplify the deployment and management of blockchain resources. It leverages Kubernetes' powerful API and integration capabilities to provide a production-ready deployment method for smart contracts and related blockchain components.

## Capabilities

- **Smart Contract Deployment**: Automate the deployment of smart contracts on various blockchain networks.
- **Blockchain Resources Management**: Manage resources like RPC providers, block explorers, networks, and wallets.
- **Version Management**: Manage multiple versions of your smart contracts.
- **Multi-Network Deployment**: Deploy your smart contracts to multiple blockchain networks at the same time.
- **Local Testing**: Use Kontract to test your smart contracts locally on your Kubernetes cluster.
- **Endless Integrations**: Seamless integration with the Kubernetes ecosystem including ci/cd pipelines, secret management, and more.

## Installation Guide

### Prerequisites

- Kubernetes cluster (version 1.18+)
- Helm (version 3+)

### Installation

1. **Clone the Git Repository**

   ```bash
   git clone https://github.com/expedio-xyz/kontract.git
   ```

2. **Install the Kontract Operator**

   Install the Kontract operator in your Kubernetes cluster:

   ```bash
   helm install kontract ./helm-chart --namespace kontract --create-namespace
   ```

   This command installs the Kontract operator in the `kontract` namespace.

## Getting Started Guide

### Basic Resources

To get started with Kontract, you'll need to define some basic resources. Below are examples of how to create a wallet, RPC provider, block explorer, network, and a simple contract.

#### Wallet

```yaml
apiVersion: kontract.expedio.xyz/v1alpha1
kind: Wallet
metadata:
  name: my-wallet
spec:
  walletType: EOA
  networkRef: ethereum-mainnet
```

#### RPCProvider

```yaml
apiVersion: kontract.expedio.xyz/v1alpha1
kind: RPCProvider
metadata:
  name: infura-mainnet
spec:
  providerName: Infura
  secretRef:
    name: infura-mainnet-api-secret
    tokenKey: key
    urlKey: endpoint
```

#### BlockExplorer

```yaml
apiVersion: kontract.expedio.xyz/v1alpha1
kind: BlockExplorer
metadata:
  name: etherscan
spec:
  explorerName: Etherscan
  secretRef:
    name: etherscan-api-secret
    tokenKey: key
    urlKey: endpoint
```

#### Network

```yaml
apiVersion: kontract.expedio.xyz/v1alpha1
kind: Network
metadata:
  name: ethereum-mainnet
spec:
  networkName: Ethereum Mainnet
  chainID: 1
  rpcProviderRef:
    name: infura-mainnet
  blockExplorerRef:
    name: etherscan
```

#### Simple Contract

```yaml
apiVersion: kontract.expedio.xyz/v1alpha1
kind: Contract
metadata:
  name: simple-contract
spec:
  contractName: SimpleContract
  networkRefs:
    - ethereum-mainnet
  walletRef: my-wallet
  code: |
    // SPDX-License-Identifier: MIT
    pragma solidity ^0.8.9;
    contract SimpleContract {
      uint128 public value;
      function setValue(uint128 newValue) public {
        value = newValue;
      }
    }
```

### Deploying the Contract

Once you have defined your resources, you can deploy the contract by applying the YAML files to your Kubernetes cluster:

```bash
kubectl apply -f wallet.yaml
kubectl apply -f rpcprovider.yaml
kubectl apply -f blockexplorer.yaml
kubectl apply -f network.yaml
kubectl apply -f simple-contract.yaml
```

This will trigger the Kontract operator to deploy your smart contract on the specified blockchain network.

## Conclusion

The Kontract operator provides a robust and flexible solution for managing blockchain resources and deploying smart contracts within a Kubernetes environment. By following this guide, you can quickly get started with deploying and managing your blockchain applications.
