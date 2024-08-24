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

// ContractProxySpec defines the desired state of ContractProxy
type ContractProxySpec struct {
	// ProxyType defines the type of proxy (e.g., Transparent)
	ProxyType string `json:"proxyType"`

	// NetworkRef references the Network resource where this proxy is deployed
	NetworkRef string `json:"networkRef"`

	// WalletRef references the Wallet resource that will sign transactions
	WalletRef string `json:"walletRef"`

	// GasStrategyRef references the GasStrategy resource for gas price management
	GasStrategyRef string `json:"gasStrategyRef"`

	// ImplementationRef references the implementation contract
	ImplementationRef string `json:"implementationRef"`

	// ProxyAdminRef references the ProxyAdmin resource managing this proxy
	ProxyAdminRef string `json:"proxyAdminRef"`
}

// ContractProxyStatus defines the observed state of ContractProxy
type ContractProxyStatus struct {
	// ProxyAddress is the address of the proxy contract on the blockchain
	ProxyAddress string `json:"proxyAddress,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ContractProxy is the Schema for the contractproxies API
type ContractProxy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ContractProxySpec   `json:"spec,omitempty"`
	Status ContractProxyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ContractProxyList contains a list of ContractProxy
type ContractProxyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ContractProxy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ContractProxy{}, &ContractProxyList{})
}
