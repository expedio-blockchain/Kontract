# Kontract Operator

Kontract is a Kubernetes operator designed to simplify the deployment and management of blockchain resources. It leverages Kubernetes' powerful API and integration capabilities to provide a production-ready deployment method for smart contracts and related blockchain components.

Kontract Dev Channel: [@KontractDev](https://t.me/+qA4EYGaHVL44MzI0)  
Kontract Support Channel: [@KontractSupport](https://t.me/+EV-Qmwxfp2M0NWJk)

![kontract-high-resolution-logo-transparent](https://github.com/user-attachments/assets/77ee4547-0c98-4b25-ace2-6670773cdb5d)

## Table of Contents

- [Capabilities](#capabilities)
- [Installation Guide](#installation-guide)
  - [Prerequisites](#prerequisites)
  - [Installation](#installation)
- [Getting Started Guide](#getting-started-guide)
  - [Local Deployment with Anvil](#local-deployment-with-anvil)
  - [Deploying the Contract](#deploying-the-contract)
  - [Observe the Contract Deployment](#observe-the-contract-deployment)
  - [Remote Blockchain Deployment](#remote-blockchain-deployment)
- [Advanced Features Guide](#advanced-features-guide)
  - [Wallet Import](#wallet-import)
  - [Adding Tests](#adding-tests)
  - [Script Deployment](#script-deployment)
  - [Multi-Network Deployment](#multi-network-deployment)
  - [Deployment with Custom Foundry Configuration](#deployment-with-custom-foundry-configuration)
  - [Deployment with Local and External Modules Import](#deployment-with-local-and-external-modules-import)
  - [Deployment with ConfigMap Code](#deployment-with-configmap-code)
  - [Initialization Parameters](#initialization-parameters)

## Capabilities

- **Smart Contract Deployment**: Automate the deployment of smart contracts on various blockchain networks.
- **Blockchain Resources Management**: Manage resources like RPC providers, block explorers, networks, and wallets.
- **Version Management**: Manage multiple versions of your smart contracts.
- **Multi-Network Deployment**: Deploy your smart contracts to multiple blockchain networks at the same time.
- **Local Testing**: Use Kontract to test your smart contracts locally on your Kubernetes cluster.
- **Endless Integrations**: Seamless integration with the Kubernetes ecosystem including CI/CD pipelines, secret management, and more.

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

### Local Deployment with Anvil

To quickly get started with Kontract, you can deploy a simple contract locally using Anvil. When using anvil local network, Kontract operator will automatically create the dependent resources like RPCProvider, Network and Wallet for you.

#### Network

```yaml
apiVersion: kontract.expedio.xyz/v1alpha1
kind: Network
metadata:
  name: anvil
spec:
  networkName: anvil
  chainID: 1
  rpcProviderRef:
    name: anvil
```

#### Contract

```yaml
apiVersion: kontract.expedio.xyz/v1alpha1
kind: Contract
metadata:
  name: anvil-contract
spec:
  contractName: AnvilContract
  networkRefs:
    - anvil
  walletRef: anvil-wallet
  code: |
    // SPDX-License-Identifier: MIT
    pragma solidity ^0.8.9;
    contract AnvilContract {
      uint128 public value;
      function setValue(uint128 newValue) public {
        value = newValue;
      }
    }
```

### Deploying the Contract

Apply the YAML files to your Kubernetes cluster to deploy the contract:

```bash
kubectl apply -f sample-resources/Networks/anvil.yaml
kubectl apply -f sample-resources/Contracts/AnvilContract.yaml
```

This will trigger the Kontract operator to deploy your smart contract on the local Anvil network.

### Observe the Contract Deployment

When contract is deployed, a new ContractVersion resource will be created and contain the deployment information.

```bash
kubectl get contractversion -o yaml anvil-contract-anvil-version-1
```

```yaml
apiVersion: kontract.expedio.xyz/v1alpha1
kind: ContractVersion
metadata:
  name: anvil-contract-anvil-version-1
  namespace: default
spec:
  code: |
    // SPDX-License-Identifier: MIT
    pragma solidity ^0.8.9;
    contract AnvilContract {
      uint128 public value;
      function setValue(uint128 newValue) public {
        value = newValue;
      }
    }
  contractName: AnvilContract
  networkRef: anvil
  walletRef: anvil-wallet
status:
  contractAddress: 0x5FbDB2315678afecb367f032d93F642f64180aa3
  transactionHash: 0x4a9c19d735998d2f752c396b05831ef3af4246786d122dda49d8f8389757ed85
  deploymentTime: "2024-10-18T19:00:09Z"
  state: deployed
```

The deployment logs can be observed in the deployment job pod.

```bash
kubectl logs job/contract-deploy-anvil-contract-anvil-version-1
```

```bash
2024-10-18 18:59:02 - Init Params:
========================================
2024-10-18 18:59:02 - Deploying the contract AnvilContract...
========================================
2024-10-18 19:00:06 - forge create src/AnvilContract.sol:AnvilContract --rpc-url http://anvil-service.default.svc.cluster.local:8545 --private-key ************
No files changed, compilation skipped
Deployer: ***REMOVED***
Deployed to: 0x5FbDB2315678afecb367f032d93F642f64180aa3
Transaction hash: 0x4a9c19d735998d2f752c396b05831ef3af4246786d122dda49d8f8389757ed85
========================================
2024-10-18 19:00:06 - Deployment completed.
2024-10-18 19:00:06 - Contract Address: 0x5FbDB2315678afecb367f032d93F642f64180aa3
2024-10-18 19:00:06 - Transaction Hash: 0x4a9c19d735998d2f752c396b05831ef3af4246786d122dda49d8f8389757ed85
========================================
```

### Remote Blockchain Deployment

To deploy a smart contract on a remote blockchain network, you need to set up several resources. Here's a step-by-step guide to help you get started.

#### Initial Setup Resources

1. **Wallet**: Represents an EOA wallet on the blockchain network (you can either create a new wallet or import an existing one).

   ```yaml
   apiVersion: kontract.expedio.xyz/v1alpha1
   kind: Wallet
   metadata:
     name: dev-wallet
   spec:
     walletType: EOA
     networkRef: ethereum-mainnet
   ```

2. **RPCProvider**: Provides the endpoint to interact with the blockchain network. You need to obtain an API key from RPC api service such as Infura, Alchemy, etc.

   ```yaml
   ---
   apiVersion: kontract.expedio.xyz/v1alpha1
   kind: RPCProvider
   metadata:
     name: infura-amoy
   spec:
     providerName: Infura
     secretRef:
       name: infura-amoy-api-secret
       tokenKey: key
       urlKey: endpoint
   ---
   apiVersion: v1
   kind: Secret
   metadata:
     name: infura-amoy-api-secret
   stringData:
     endpoint: https://polygon-amoy.infura.io/v3/
     key: 39f79****************4de4b
   ```

3. **BlockExplorer**: Used to verify the contract on the exteral blockexplorer. You need to obtain an API key from the blockexplorer service such as, etherscan, polygonscan, etc.

   ```yaml
   apiVersion: kontract.expedio.xyz/v1alpha1
   kind: BlockExplorer
   metadata:
     name: polygonscan-block-explorer
   spec:
     explorerName: polygonscan
     secretRef:
       name: polygonscan-api-secret
       tokenKey: key
       urlKey: endpoint
   ```

   ```yaml
   apiVersion: v1
   kind: Secret
   metadata:
     name: polygonscan-api-secret
   stringData:
     endpoint: https://api-amoy.polygonscan.com/api
     key: JPQV****************15A9
   ```

4. **Network**: Defines the blockchain network where the contract will be deployed. It references the RPCProvider and BlockExplorer.

   ```yaml
   apiVersion: kontract.expedio.xyz/v1alpha1
   kind: Network
   metadata:
     name: amoy
   spec:
     networkName: amoy-testnet
     chainID: 18000
     rpcProviderRef:
       name: infura-amoy
     blockExplorerRef:
       name: polygonscan-block-explorer
   ```

5. **Contract**: Represents the smart contract to be deployed. It includes the contract code and references the network and wallet.

   ```yaml
   apiVersion: kontract.expedio.xyz/v1alpha1
   kind: Contract
   metadata:
     name: simple-contract
   spec:
     contractName: SimpleContract
     networkRefs:
       - amoy
     walletRef: dev-wallet
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

#### Deploying the Contract

Once you have defined your resources, apply the YAML files to your Kubernetes cluster:

```bash
kubectl apply -f sample-resources/Wallets/wallet.yaml
kubectl apply -f sample-resources/RPCProvider/InfuraAmoy.yaml
kubectl apply -f sample-resources/BlockExplorer/polygonscan.yaml
kubectl apply -f sample-resources/Networks/amoy.yaml
kubectl apply -f sample-resources/Contracts/SimpleContract.yaml
```

This will trigger the Kontract operator to deploy your smart contract on the specified remote blockchain network.

## Advanced Features Guide

### Wallet Import

You can import an existing wallet using a Kubernetes secret. This is useful for deploying contracts with a pre-existing account.

```yaml
apiVersion: kontract.expedio.xyz/v1alpha1
kind: Wallet
metadata:
  name: imported-wallet
spec:
  walletType: EOA
  networkRef: ethereum-mainnet
  importFrom:
    secretRef: wallet-secret
```

### Adding Tests

You can include tests for your smart contracts to ensure they function as expected. Tests are written in Solidity and can be included in the contract specification.

```yaml
apiVersion: kontract.expedio.xyz/v1alpha1
kind: Contract
metadata:
  name: simple-contract
spec:
  contractName: SimpleContract
  networkRefs:
    - amoy
  walletRef: dev-wallet
  code: |
    // SPDX-License-Identifier: MIT
    pragma solidity ^0.8.9;
    contract SimpleContract {
      uint128 public value;
      function setValue(uint128 newValue) public {
        value = newValue;
      }
    }
  test: |
    // SPDX-License-Identifier: MIT
    pragma solidity ^0.8.9;
    import "src/SimpleContract.sol";
    contract SimpleContractTest {
      SimpleContract myContract;
      function setUp() public {
        myContract = new SimpleContract();
      }
      function testInitialValue() public {
        myContract.setValue(1);
      }
    }
```

### Script Deployment

You can use scripts to automate complex deployment processes. This is useful for setting up contracts with specific initialization logic.

```yaml
apiVersion: kontract.expedio.xyz/v1alpha1
kind: Contract
metadata:
  name: script-contract
spec:
  contractName: ScriptContract
  networkRefs:
    - amoy
  walletRef: dev-wallet
  code: |
    // SPDX-License-Identifier: MIT
    pragma solidity ^0.8.0;
    contract ScriptContract {
      uint256 public value;
      function setValue(uint256 newValue) public {
        value = newValue;
      }
    }
  script: |
    // SPDX-License-Identifier: MIT
    pragma solidity ^0.8.0;
    import "forge-std/Script.sol";
    import "../src/ScriptContract.sol";
    contract DeploymentScript is Script {
      function run() external {
        vm.startBroadcast();
        ScriptContract myContract = new ScriptContract();
        console.log("Contract deployed at:", address(myContract));
        vm.stopBroadcast();
      }
    }
```

### Multi-Network Deployment

Deploy your contract to multiple networks simultaneously by specifying multiple network references.

```yaml
apiVersion: kontract.expedio.xyz/v1alpha1
kind: Contract
metadata:
  name: simple-contract
spec:
  contractName: SimpleContract
  networkRefs:
    - holesky
    - amoy
    - sepolia
  walletRef: dev-wallet
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

### Deployment with Custom Foundry Configuration

Customize your deployment environment using a Foundry configuration file.

```yaml
apiVersion: kontract.expedio.xyz/v1alpha1
kind: Contract
metadata:
  name: foundry-config-contract
spec:
  contractName: FoundryConfigContract
  networkRefs:
    - holesky
  walletRef: dev-wallet
  code: |
    // SPDX-License-Identifier: MIT
    pragma solidity ^0.8.9;
    contract FoundryConfigContract {
      uint128 public value;
      function setValue(uint128 newValue) public {
        value = newValue;
      }
    }
  foundryConfig: |
    [rpc_endpoints]
    mainnet = "https://eth.llamarpc.com"
    [profile.default]
    src = "src"
    out = "out"
    libs = ["lib"]
    ffi = true
    fs_permissions = [{ access = "read-write", path = ".forge-snapshots/"}]
    solc_version = "0.8.26"
    evm_version = "cancun"
    eth_rpc_url = "https://eth.llamarpc.com"
```

### Deployment with Local and External Modules Import

Import external libraries from Git and local modules from ConfigMap to extend your contract's functionality.

```yaml
---
apiVersion: kontract.expedio.xyz/v1alpha1
kind: Contract
metadata:
  name: complex-contract
spec:
  contractName: BlockchainStocks
  networkRefs:
    - holesky
  walletRef: dev-wallet
  externalModules:
    - "OpenZeppelin/openzeppelin-contracts@v4.8.3"
  localModules:
    - name: dividend
  code: |
    // SPDX-License-Identifier: Unlicense
    pragma solidity ^0.8.9;
    import "lib/openzeppelin-contracts/contracts/token/ERC1155/ERC1155.sol";
    import "src/dividend/Dividend.sol";
    contract BlockchainStocks is ERC1155, Dividend {
        // Contract code here
    }
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: dividend
data:
  Dividend.sol: |
    // SPDX-License-Identifier: Unlicense
    pragma solidity ^0.8.9;

    abstract contract Dividend {
        event DividendReleased(address to, uint256 amount);
        mapping(address => uint256) private _dividendReleased;
        uint256 private _totalDividendReleased;

        function releasedPerToken(
            uint256 _fromReleased,
            uint256 _fromBalance,
            uint256 _amountTransfered
        ) internal pure returns (uint256) {
            uint256 _releasedPerToken = _fromReleased /
                (_amountTransfered + _fromBalance) +
                1;
            return _releasedPerToken;
        }
```

### Deployment with ConfigMap Code

Store your contract code in a Kubernetes ConfigMap for easy management and updates.

```yaml
apiVersion: kontract.expedio.xyz/v1alpha1
kind: Contract
metadata:
  name: configmap-based-contract
spec:
  contractName: ConfigMapBasedContract
  networkRefs:
    - holesky
  walletRef: dev-wallet
  codeRef:
    name: contract-code-configmap
    key: code
  testRef:
    name: contract-test-configmap
    key: test
  foundryConfigRef:
    name: foundry-config-configmap
    key: foundry.toml
```

### Initialization Parameters

Initialization parameters allow you to pass specific values to your contract's constructor during deployment. This is useful for setting initial states or configurations.

```yaml
apiVersion: kontract.expedio.xyz/v1alpha1
kind: Contract
metadata:
  name: complex-contract
spec:
  contractName: BlockchainStocks
  networkRefs:
    - holesky
    - amoy
    - sepolia
  walletRef: dev-wallet
  initParams:
    - "https://expedio.xyz/metadata"
    - "0x1234567890abcdef1234567890abcdef12345678"
    - "100000"
    - "1000000000000000000"
  code: |
    // SPDX-License-Identifier: Unlicense
    pragma solidity ^0.8.9;
    import "lib/openzeppelin-contracts/contracts/token/ERC1155/ERC1155.sol";
    import "src/dividend/Dividend.sol";
    contract BlockchainStocks is ERC1155, Dividend {
        // Contract code here
    }
```
