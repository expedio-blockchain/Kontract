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

// ActionParameter represents a parameter to be passed to the contract function
type ActionParameter struct {
	// Name is the name of the parameter
	Name string `json:"name"`

	// Value is the value of the parameter
	Value string `json:"value"`
}

// ActionSpec defines the desired state of Action
type ActionSpec struct {
	// ActionType defines the type of action (e.g., invoke, query, upgrade, test)
	ActionType string `json:"actionType"`

	// ContractRef references the Contract resource for the action
	ContractRef string `json:"contractRef"`

	// WalletRef references the Wallet resource used for the action
	WalletRef string `json:"walletRef"`

	// NetworkRef references the Network resource where the action will be executed
	NetworkRef string `json:"networkRef"`

	// GasStrategyRef references the GasStrategy resource for gas price management
	GasStrategyRef string `json:"gasStrategyRef"`

	// FunctionName is the name of the contract function to execute (for invoke or test actions)
	FunctionName string `json:"functionName,omitempty"`

	// Parameters are the parameters to pass to the contract function
	Parameters []ActionParameter `json:"parameters,omitempty"`

	// Schedule is an optional cron schedule for recurring actions
	Schedule string `json:"schedule,omitempty"`
}

// ActionStatus defines the observed state of Action
type ActionStatus struct {
	// LastExecution is the timestamp of the last execution of the action
	LastExecution metav1.Time `json:"lastExecution,omitempty"`

	// TransactionHash is the transaction hash of the last action execution (if applicable)
	TransactionHash string `json:"transactionHash,omitempty"`

	// Result is the result of the last action execution (e.g., Success, Failure)
	Result string `json:"result,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Action is the Schema for the actions API
type Action struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ActionSpec   `json:"spec,omitempty"`
	Status ActionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ActionList contains a list of Action
type ActionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Action `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Action{}, &ActionList{})
}
