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

// RPCProviderSpec defines the desired state of RPCProvider
type RPCProviderSpec struct {
	// ProviderName is the name of the RPC provider (e.g., Infura)
	ProviderName string `json:"providerName"`

	// Endpoint is the URL of the RPC provider
	Endpoint string `json:"endpoint"`

	// SecretRef references a Kubernetes Secret that contains the API token
	SecretRef corev1.SecretReference `json:"secretRef"`

	// Timeout defines the request timeout for the RPC calls
	Timeout metav1.Duration `json:"timeout"`
}

// RPCProviderStatus defines the observed state of RPCProvider
type RPCProviderStatus struct {
	// Add status fields here if needed
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// RPCProvider is the Schema for the rpcproviders API
type RPCProvider struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RPCProviderSpec   `json:"spec,omitempty"`
	Status RPCProviderStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RPCProviderList contains a list of RPCProvider
type RPCProviderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RPCProvider `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RPCProvider{}, &RPCProviderList{})
}
