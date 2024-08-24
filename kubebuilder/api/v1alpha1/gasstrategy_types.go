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

// GasStrategySpec defines the desired state of GasStrategy
type GasStrategySpec struct {
	// StrategyType specifies the type of gas strategy (e.g., fixed, oracle, dynamic)
	StrategyType string `json:"strategyType"`

	// GasPriceOracle is the URL to the gas price oracle service
	GasPriceOracle string `json:"gasPriceOracle,omitempty"`

	// FallbackGasPrice is the gas price to use if the oracle is unavailable
	FallbackGasPrice string `json:"fallbackGasPrice,omitempty"`

	// MaxGasPrice is the maximum allowed gas price
	MaxGasPrice string `json:"maxGasPrice,omitempty"`

	// MinGasPrice is the minimum allowed gas price
	MinGasPrice string `json:"minGasPrice,omitempty"`

	// SecretRef references the Kubernetes Secret that contains the API token for the oracle
	SecretRef *corev1.SecretReference `json:"secretRef,omitempty"`
}

// GasStrategyStatus defines the observed state of GasStrategy
type GasStrategyStatus struct {
	// Add status fields here if needed
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// GasStrategy is the Schema for the gasstrategies API
type GasStrategy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GasStrategySpec   `json:"spec,omitempty"`
	Status GasStrategyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GasStrategyList contains a list of GasStrategy
type GasStrategyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GasStrategy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GasStrategy{}, &GasStrategyList{})
}
