## Advanced Features Guide

In this section, you'll find examples of specific use cases to better adjust Kontract to your unique requirements.

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

Automate complex deployment processes using scripts. This is useful for setting up contracts with specific initialization logic.

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

Deploy your contract to multiple networks at once by specifying multiple network references.

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

Manage and update your contract code easily by storing it in a Kubernetes ConfigMap.

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

## What's Next?

Join the community!

- [Official Telegram Channel](https://t.me/kontract_support)
- [Feature Requests & Contribute Code](https://github.com/expedio-blockchain/Kontract/issues)
