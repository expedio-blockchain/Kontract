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

// BlockExplorerSpec defines the desired state of BlockExplorer
type BlockExplorerSpec struct {
	// ExplorerName is the name of the block explorer (e.g., Etherscan)
	ExplorerName string `json:"explorerName"`

	// SecretRef references a Kubernetes Secret and specifies the keys for API token and endpoint
	SecretRef BlockExplorerSecretRef `json:"secretRef"`
}

// BlockExplorerSecretRef is a custom struct for handling specific keys in the Secret
type BlockExplorerSecretRef struct {
	Name        string `json:"name"`
	APIKey      string `json:"apiKey"`
	APIEndpoint string `json:"apiEndpoint"`
}

// BlockExplorerStatus defines the observed state of BlockExplorer
type BlockExplorerStatus struct {
	Healthy     bool   `json:"healthy,omitempty"`
	APIEndpoint string `json:"apiEndpoint,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// BlockExplorer is the Schema for the blockexplorers API
type BlockExplorer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BlockExplorerSpec   `json:"spec,omitempty"`
	Status BlockExplorerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BlockExplorerList contains a list of BlockExplorer
type BlockExplorerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BlockExplorer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BlockExplorer{}, &BlockExplorerList{})
}
