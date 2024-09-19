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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ContractVersionSpec defines the desired state of ContractVersion
type ContractVersionSpec struct {
	ContractName    string               `json:"contractName"`
	NetworkRef      string               `json:"networkRef"`
	WalletRef       string               `json:"walletRef"`
	GasStrategyRef  string               `json:"gasStrategyRef"`
	Code            string               `json:"code"`
	Test            string               `json:"test,omitempty"`
	InitParams      []string             `json:"initParams,omitempty"`
	ExternalModules []string             `json:"externalModules,omitempty"`
	LocalModules    []ConfigMapReference `json:"localModules,omitempty"`
	Script          string               `json:"script,omitempty"`
}

// ContractVersionStatus defines the observed state of ContractVersion
type ContractVersionStatus struct {
	ContractAddress string      `json:"contractAddress,omitempty"`
	DeploymentTime  metav1.Time `json:"deploymentTime,omitempty"`
	TransactionHash string      `json:"transactionHash,omitempty"`
	Test            string      `json:"test,omitempty"`
	State           string      `json:"state,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ContractVersion is the Schema for the contractversions API
type ContractVersion struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ContractVersionSpec   `json:"spec,omitempty"`
	Status ContractVersionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ContractVersionList contains a list of ContractVersion
type ContractVersionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ContractVersion `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ContractVersion{}, &ContractVersionList{})
}
