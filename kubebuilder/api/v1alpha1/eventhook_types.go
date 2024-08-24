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

// EventFilter represents optional conditions to filter events
type EventFilter struct {
	// BlockNumber is the block number to filter by
	BlockNumber string `json:"blockNumber,omitempty"`

	// EventName is the name of the event to filter by
	EventName string `json:"eventName,omitempty"`
}

// EventHookSpec defines the desired state of EventHook
type EventHookSpec struct {
	// EventType defines the event that triggers the hook (e.g., BlockMined, ContractEvent)
	EventType string `json:"eventType"`

	// ContractRef references the Contract resource that the event relates to
	ContractRef string `json:"contractRef"`

	// ActionRef references the Action resource to be triggered
	ActionRef string `json:"actionRef"`

	// Filter specifies optional conditions to filter events
	Filter EventFilter `json:"filter,omitempty"`
}

// EventHookStatus defines the observed state of EventHook
type EventHookStatus struct {
	// Add status fields here if needed
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// EventHook is the Schema for the eventhooks API
type EventHook struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EventHookSpec   `json:"spec,omitempty"`
	Status EventHookStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EventHookList contains a list of EventHook
type EventHookList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EventHook `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EventHook{}, &EventHookList{})
}
