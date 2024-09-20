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
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kontractdeployerv1alpha1 "github.com/expedio-blockchain/KontractDeployer/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NetworkReconciler reconciles a Network object
type NetworkReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	EventRecorder record.EventRecorder
}

// +kubebuilder:rbac:groups=kontractdeployer.expedio.xyz,resources=networks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kontractdeployer.expedio.xyz,resources=networks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kontractdeployer.expedio.xyz,resources=networks/finalizers,verbs=update

// Reconcile is part of the main Kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *NetworkReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the Network instance
	var network kontractdeployerv1alpha1.Network
	if err := r.Get(ctx, req.NamespacedName, &network); err != nil {
		logger.Error(err, "unable to fetch Network")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Handle Anvil network creation
	if network.Spec.NetworkName == "anvil" {
		if err := r.createAnvilResources(ctx, &network); err != nil {
			logger.Error(err, "failed to create Anvil resources")
			return ctrl.Result{}, err
		}
		if err := r.createAnvilWallet(ctx, &network); err != nil {
			logger.Error(err, "failed to create Anvil wallet")
			return ctrl.Result{}, err
		}
	}

	// Fetch the referenced RPCProvider
	var rpcProvider kontractdeployerv1alpha1.RPCProvider
	rpcProviderKey := client.ObjectKey{Name: network.Spec.RPCProviderRef.Name, Namespace: network.Namespace}
	if err := r.Get(ctx, rpcProviderKey, &rpcProvider); err != nil {
		logger.Error(err, "unable to fetch RPCProvider", "RPCProviderKey", rpcProviderKey)
		r.EventRecorder.Event(&network, "Warning", "MissingRPCProvider", "RPCProvider is specified but missing")
		return ctrl.Result{}, err
	}

	// Fetch the referenced BlockExplorer if it exists
	if network.Spec.BlockExplorerRef != nil {
		var blockExplorer kontractdeployerv1alpha1.BlockExplorer
		blockExplorerKey := client.ObjectKey{Name: network.Spec.BlockExplorerRef.Name, Namespace: network.Namespace}
		if err := r.Get(ctx, blockExplorerKey, &blockExplorer); err != nil {
			logger.Error(err, "unable to fetch BlockExplorer", "BlockExplorerKey", blockExplorerKey)
			r.EventRecorder.Event(&network, "Warning", "MissingBlockExplorer", "BlockExplorer is specified but missing")
			return ctrl.Result{}, err
		}
		network.Status.BlockExplorerEndpoint = blockExplorer.Status.APIEndpoint
		network.Status.Healthy = rpcProvider.Status.Healthy && blockExplorer.Status.Healthy
	} else {
		network.Status.Healthy = rpcProvider.Status.Healthy
	}

	// Update Network status with RPC endpoint
	network.Status.RPCEndpoint = rpcProvider.Status.APIEndpoint

	// Update the Network status
	if err := r.Status().Update(ctx, &network); err != nil {
		logger.Error(err, "unable to update Network status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// createAnvilResources creates a Pod and Service for the Anvil network
func (r *NetworkReconciler) createAnvilResources(ctx context.Context, network *kontractdeployerv1alpha1.Network) error {
	logger := log.FromContext(ctx)

	// Define the Anvil Pod
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "anvil-pod",
			Namespace: network.Namespace,
			Labels:    map[string]string{"app": "anvil"},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "anvil",
					Image:   "docker.io/expedio/foundry:latest",
					Command: []string{"anvil", "--chain-id", fmt.Sprintf("%d", network.Spec.ChainID)},
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 8545,
							HostPort:      8545,
						},
					},
				},
			},
		},
	}

	// Create the Pod
	if err := r.Create(ctx, pod); err != nil {
		logger.Error(err, "failed to create Anvil Pod")
		return err
	}

	// Define the Anvil Service
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "anvil-service",
			Namespace: network.Namespace,
			Labels:    map[string]string{"app": "anvil"},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "anvil"},
			Ports: []corev1.ServicePort{
				{
					Port:     8545,
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}

	// Create the Service
	if err := r.Create(ctx, service); err != nil {
		logger.Error(err, "failed to create Anvil Service")
		return err
	}

	// Create the RPCProvider and Secret for Anvil
	if err := r.createAnvilRPCProvider(ctx, network); err != nil {
		logger.Error(err, "failed to create Anvil RPCProvider and Secret")
		return err
	}

	return nil
}

// createAnvilRPCProvider creates an RPCProvider and Secret for the Anvil network
func (r *NetworkReconciler) createAnvilRPCProvider(ctx context.Context, network *kontractdeployerv1alpha1.Network) error {
	logger := log.FromContext(ctx)

	// Define the Secret
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "anvil-rpc-secret",
			Namespace: network.Namespace,
		},
		StringData: map[string]string{
			"tokenKey": "",
			"urlKey":   "http://anvil-service:8545",
		},
	}

	// Create the Secret
	if err := r.Create(ctx, secret); err != nil {
		logger.Error(err, "failed to create Anvil RPC Secret")
		return err
	}

	// Define the RPCProvider
	rpcProvider := &kontractdeployerv1alpha1.RPCProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "anvil-rpc-provider",
			Namespace: network.Namespace,
		},
		Spec: kontractdeployerv1alpha1.RPCProviderSpec{
			ProviderName: "Anvil",
			SecretRef: kontractdeployerv1alpha1.SecretKeyReference{
				Name:     "anvil-rpc-secret",
				TokenKey: "tokenKey",
				URLKey:   "urlKey",
			},
		},
	}

	// Create the RPCProvider
	if err := r.Create(ctx, rpcProvider); err != nil {
		logger.Error(err, "failed to create Anvil RPCProvider")
		return err
	}

	return nil
}

// createAnvilWallet creates a Wallet and Secret for the Anvil network
func (r *NetworkReconciler) createAnvilWallet(ctx context.Context, network *kontractdeployerv1alpha1.Network) error {
	logger := log.FromContext(ctx)

	// Define the Secret for the Anvil wallet
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "anvil-wallet-secret",
			Namespace: network.Namespace,
		},
		StringData: map[string]string{
			"privateKey": "***REMOVED***",
			"publicKey":  "***REMOVED***",
		},
	}

	// Create the Secret
	if err := r.Create(ctx, secret); err != nil {
		logger.Error(err, "failed to create Anvil Wallet Secret")
		return err
	}

	// Define the Wallet
	wallet := &kontractdeployerv1alpha1.Wallet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "anvil-wallet",
			Namespace: network.Namespace,
		},
		Spec: kontractdeployerv1alpha1.WalletSpec{
			WalletType: "EOA",
			NetworkRef: network.Name,
			ImportFrom: &kontractdeployerv1alpha1.ImportFromSpec{
				SecretRef: "anvil-wallet-secret",
			},
		},
		Status: kontractdeployerv1alpha1.WalletStatus{
			PublicKey: "***REMOVED***",
			SecretRef: "anvil-wallet-secret",
		},
	}

	// Create the Wallet
	if err := r.Create(ctx, wallet); err != nil {
		logger.Error(err, "failed to create Anvil Wallet")
		return err
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NetworkReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kontractdeployerv1alpha1.Network{}).
		Complete(r)
}
