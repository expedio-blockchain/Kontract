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

package controller

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kontractdeployerv1alpha1 "github.com/expedio-blockchain/KontractDeployer/api/v1alpha1"
)

// WalletReconciler reconciles a Wallet object
type WalletReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=kontractdeployer.expedio.xyz,resources=wallets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kontractdeployer.expedio.xyz,resources=wallets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kontractdeployer.expedio.xyz,resources=wallets/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=create;update;get;list;watch

// Reconcile is part of the main Kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *WalletReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the Wallet instance
	wallet := &kontractdeployerv1alpha1.Wallet{}
	err := r.Get(ctx, req.NamespacedName, wallet)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if the wallet is already created
	if wallet.Status.PublicKey != "" && wallet.Status.SecretRef != "" {
		logger.Info("Wallet already created", "PublicKey", wallet.Status.PublicKey)
		return ctrl.Result{}, nil
	}

	var secretName string
	if wallet.Spec.ImportFrom != nil && wallet.Spec.ImportFrom.SecretRef != "" {
		// Use the provided secretRef
		secretName = wallet.Spec.ImportFrom.SecretRef
		logger.Info("Using existing secret", "SecretRef", secretName)

		// Fetch the Secret to extract the public key
		existingSecret := &corev1.Secret{}
		err := r.Get(ctx, client.ObjectKey{Name: secretName, Namespace: req.Namespace}, existingSecret)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to fetch existing secret: %v", err)
		}

		// Extract the public key from the Secret
		publicKey, exists := existingSecret.Data["publicKey"]
		if !exists {
			return ctrl.Result{}, fmt.Errorf("publicKey not found in the secret: %s", secretName)
		}

		// Update the Wallet status with the public key and secretRef
		wallet.Status.PublicKey = string(publicKey)
		wallet.Status.SecretRef = secretName
		err = r.Status().Update(ctx, wallet)
		if err != nil {
			return ctrl.Result{}, err
		}
		logger.Info("Wallet imported", "PublicKey", wallet.Status.PublicKey, "SecretRef", wallet.Status.SecretRef)

	} else {
		// Generate a new Ethereum wallet
		privateKey, err := crypto.GenerateKey()
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to generate wallet: %v", err)
		}

		publicKey := crypto.PubkeyToAddress(privateKey.PublicKey).Hex()
		privateKeyHex := hex.EncodeToString(crypto.FromECDSA(privateKey))

		// Create a new Kubernetes Secret to store the wallet keys
		secretName = fmt.Sprintf("%s-wallet-secret", wallet.Name)
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: req.Namespace,
			},
			StringData: map[string]string{
				"privateKey": privateKeyHex,
				"publicKey":  publicKey,
			},
		}

		// Set the Wallet instance as the owner of the Secret
		if err := controllerutil.SetControllerReference(wallet, secret, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}

		// Check if the Secret already exists
		found := &corev1.Secret{}
		err = r.Get(ctx, client.ObjectKey{Name: secret.Name, Namespace: secret.Namespace}, found)
		if err != nil && client.IgnoreNotFound(err) == nil {
			// Create the Secret
			err = r.Create(ctx, secret)
			if err != nil {
				return ctrl.Result{}, err
			}
			logger.Info("Secret created", "Secret.Name", secret.Name)
		} else if err != nil {
			return ctrl.Result{}, err
		}

		// Update the Wallet status with the public key and secretRef
		wallet.Status.PublicKey = publicKey
		wallet.Status.SecretRef = secretName
		err = r.Status().Update(ctx, wallet)
		if err != nil {
			return ctrl.Result{}, err
		}

		logger.Info("Wallet created", "PublicKey", publicKey, "SecretRef", secretName)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WalletReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kontractdeployerv1alpha1.Wallet{}).
		Complete(r)
}
