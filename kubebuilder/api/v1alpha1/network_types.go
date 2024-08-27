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

// NetworkSpec defines the desired state of Network
type NetworkSpec struct {
	// NetworkName is the name of the blockchain network (e.g., EthereumMainnet)
	NetworkName string `json:"networkName"`

	// ChainID is the unique identifier for the blockchain network (e.g., 1 for Ethereum Mainnet)
	ChainID int `json:"chainID"`

	// RPCProviderRef references the RPCProvider resource to be used for interacting with the blockchain
	RPCProviderRef corev1.LocalObjectReference `json:"rpcProviderRef"`

	// BlockExplorerRef references the BlockExplorer resource to be used for querying blockchain data
	BlockExplorerRef corev1.LocalObjectReference `json:"blockExplorerRef"`
}

// NetworkStatus defines the observed state of Network
type NetworkStatus struct {
	// RPCEndpoint is the endpoint URL for the RPC provider
	RPCEndpoint string `json:"rpcEndpoint,omitempty"`

	// BlockExplorerEndpoint is the endpoint URL for the Block Explorer
	BlockExplorerEndpoint string `json:"blockExplorerEndpoint,omitempty"`

	// Healthy indicates whether the network is healthy
	Healthy bool `json:"healthy,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Network is the Schema for the networks API
type Network struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkSpec   `json:"spec,omitempty"`
	Status NetworkStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NetworkList contains a list of Network
type NetworkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Network `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Network{}, &NetworkList{})
}
