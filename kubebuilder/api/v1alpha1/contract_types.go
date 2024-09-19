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

// ConfigMapReference defines a reference to a ConfigMap
type ConfigMapReference struct {
	// Name of the ConfigMap
	Name string `json:"name"`
}

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

	// NetworkRefs references the Network resources where this contract is deployed
	NetworkRefs []string `json:"networkRefs"`

	// WalletRef references the Wallet resource that will sign transactions
	WalletRef string `json:"walletRef"`

	// GasStrategyRef references the GasStrategy resource for gas price management
	GasStrategyRef string `json:"gasStrategyRef"`

	// ExternalModules is a list of external modules to be imported via npm
	ExternalModules []string `json:"externalModules,omitempty"`

	// LocalModules is a list of local modules to be imported from ConfigMap
	LocalModules []ConfigMapReference `json:"localModules,omitempty"`

	// Code is the source code of the smart contract
	Code string `json:"code"`

	// Test is the source code for testing the smart contract
	// Optional
	Test string `json:"test,omitempty"`

	// InitParams is a list of initialization parameters for the contract
	InitParams []string `json:"initParams,omitempty"`

	// Script is the source code of the deployment script
	// Optional
	Script string `json:"script,omitempty"`
}

// ContractStatus defines the observed state of Contract
type ContractStatus struct {
	// CurrentVersion is the current version of the contract
	CurrentVersion string `json:"currentVersion,omitempty"`
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
