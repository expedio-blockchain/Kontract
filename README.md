# Kontract Operator

Kontract is a Kubernetes operator designed to simplify the deployment and management of blockchain resources. It leverages Kubernetes' powerful API and integration capabilities to provide a production-ready deployment method for smart contracts and related blockchain components.

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

### Simple Local Deployment with Anvil

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
  test: |
    // SPDX-License-Identifier: MIT
    pragma solidity ^0.8.9;
    import "src/AnvilContract.sol";
    contract AnvilContractTest {
      AnvilContract myContract;
      function setUp() public {
        myContract = new AnvilContract();
      }
      function testInitialValue() public {
        myContract.setValue(1);
      }
    }
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
2024-10-18 18:59:02 - Running tests...
Compiling 2 files with Solc 0.8.27
Solc 0.8.27 finished in 17.43ms
Compiler run successful!

Ran 1 test for test/AnvilContract.t.sol:AnvilContractTest
[PASS] testInitialValue() (gas: 27531)
Suite result: ok. 1 passed; 0 failed; 0 skipped; finished in 2.26ms (306.50µs CPU time)

Ran 1 test suite in 5.17ms (2.26ms CPU time): 1 tests passed, 0 failed, 0 skipped (1 total tests)
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

## Advanced Documentation

### Setting Up Basic Resources

Below are examples of how to create a wallet, RPC provider, block explorer, network, and a simple contract.

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

### Exploring Specific Features

#### Wallet Import

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

#### Contract with External and Local Modules

```yaml
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
```

#### Script Contract Deployment

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

#### ConfigMap Contract Code Resource

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
```

#### Custom Foundry Config Contract

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

## Conclusion

The Kontract operator provides a robust and flexible solution for managing blockchain resources and deploying smart contracts within a Kubernetes environment. By following this guide, you can quickly get started with deploying and managing your blockchain applications, and explore advanced features for more complex use cases.

```

```
