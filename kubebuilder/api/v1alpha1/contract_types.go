/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ContractSpec defines the desired state of Contract
type ContractSpec struct {
	// Import indicates whether the contract should be imported (true) or deployed (false)
	// +kubebuilder:default=false
	Import bool `json:"import,omitempty"`

	// ImportContractAddress is the address of the contract to be imported
	// Only required if Import is true
	ImportContractAddress string `json:"importContractAddress,omitempty"`

	// ContractName is the name of the smart contract
	ContractName string `json:"contractName"`

	// NetworkRef references the Network resource where this contract is deployed
	NetworkRef string `json:"networkRef"`

	// WalletRef references the Wallet resource that will sign transactions
	WalletRef string `json:"walletRef"`

	// GasStrategyRef references the GasStrategy resource for gas price management
	GasStrategyRef string `json:"gasStrategyRef"`

	// Code is the source code of the smart contract
	Code string `json:"code"`

	// Test is the source code for testing the smart contract
	// Optional
	Test string `json:"test,omitempty"`
}

// ContractStatus defines the observed state of Contract
type ContractStatus struct {
	// ContractAddress is the address of the deployed contract
	ContractAddress string `json:"contractAddress,omitempty"`

	// DeploymentTime is the timestamp when the contract was deployed
	DeploymentTime metav1.Time `json:"deploymentTime,omitempty"`

	// TransactionHash is the hash of the transaction that deployed the contract
	TransactionHash string `json:"transactionHash,omitempty"`

	// Test indicates the result of the contract tests (e.g., passed, failed)
	Test string `json:"test,omitempty"`

	// State represents the current state of the contract (e.g., deployed, failed)
	State string `json:"state,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Contract is the Schema for the contracts API
type Contract struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ContractSpec   `json:"spec,omitempty"`
	Status ContractStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ContractList contains a list of Contract
type ContractList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Contract `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Contract{}, &ContractList{})
}
