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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProxyAdminSpec defines the desired state of ProxyAdmin
type ProxyAdminSpec struct {
	// NetworkRef references the Network resource where this ProxyAdmin is used
	NetworkRef string `json:"networkRef"`

	// WalletRef references the Wallet resource that will sign transactions
	WalletRef string `json:"walletRef"`

	// GasStrategyRef references the GasStrategy resource for gas price management
	GasStrategyRef string `json:"gasStrategyRef"`

	// AdminAddress is the address of the admin contract on the blockchain
	AdminAddress string `json:"adminAddress"`
}

// ProxyAdminStatus defines the observed state of ProxyAdmin
type ProxyAdminStatus struct {
	// ContractProxyRefs lists the proxies managed by this ProxyAdmin
	ContractProxyRefs []corev1.LocalObjectReference `json:"contractProxyRefs,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ProxyAdmin is the Schema for the proxyadmins API
type ProxyAdmin struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProxyAdminSpec   `json:"spec,omitempty"`
	Status ProxyAdminStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProxyAdminList contains a list of ProxyAdmin
type ProxyAdminList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProxyAdmin `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ProxyAdmin{}, &ProxyAdminList{})
}
