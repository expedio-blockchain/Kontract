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

// WalletSpec defines the desired state of Wallet
type WalletSpec struct {
	// WalletType specifies the type of wallet (e.g., EOA, Contract)
	WalletType string `json:"walletType"`

	// SecretRef references a Kubernetes Secret that contains the wallet's private key or mnemonic
	SecretRef string `json:"secretRef"`

	// NetworkRef references the Network resource where this wallet is used
	NetworkRef string `json:"networkRef"`

	// Import indicates whether the wallet should be imported from an existing secret
	Import bool `json:"import"`
}

// WalletStatus defines the observed state of Wallet
type WalletStatus struct {
	// PublicKey stores the public key associated with the wallet
	PublicKey string `json:"publicKey"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Wallet is the Schema for the wallets API
type Wallet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WalletSpec   `json:"spec,omitempty"`
	Status WalletStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WalletList contains a list of Wallet
type WalletList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Wallet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Wallet{}, &WalletList{})
}
