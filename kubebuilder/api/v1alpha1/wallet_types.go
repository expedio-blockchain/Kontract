package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ImportFromSpec defines the optional import settings
type ImportFromSpec struct {
	// SecretRef references a Kubernetes Secret that contains the wallet's private key or mnemonic
	SecretRef string `json:"secretRef,omitempty"`
}

// WalletSpec defines the desired state of Wallet
type WalletSpec struct {
	// WalletType specifies the type of wallet (e.g., EOA, Contract)
	WalletType string `json:"walletType"`

	// NetworkRef references the Network resource where this wallet is used
	NetworkRef string `json:"networkRef"`

	// ImportFrom specifies the details for importing an existing wallet
	ImportFrom *ImportFromSpec `json:"importFrom,omitempty"`
}

// WalletStatus defines the observed state of Wallet
type WalletStatus struct {
	// PublicKey stores the public key associated with the wallet
	PublicKey string `json:"publicKey"`

	// SecretRef stores the reference to the Kubernetes Secret that contains the wallet's private key or mnemonic
	SecretRef string `json:"secretRef"`
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
