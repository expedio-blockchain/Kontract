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

// Grant permissions to create Pods
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods/status,verbs=get;update;patch

// Grant permissions to create Services
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete

// Grant permissions to create Secrets
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Grant permissions to manage Wallets
// +kubebuilder:rbac:groups=kontractdeployer.expedio.xyz,resources=wallets,verbs=get;list;watch;create;update;patch;delete

// Grant permissions to manage RPCProviders
// +kubebuilder:rbac:groups=kontractdeployer.expedio.xyz,resources=rpcproviders,verbs=get;list;watch;create;update;patch;delete

const networkFinalizer = "kontractdeployer.expedio.xyz/network-finalizer"

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

	// Check if the Network instance is marked for deletion
	if network.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it doesn't have our finalizer, add it
		if !containsString(network.ObjectMeta.Finalizers, networkFinalizer) {
			network.ObjectMeta.Finalizers = append(network.ObjectMeta.Finalizers, networkFinalizer)
			if err := r.Update(ctx, &network); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if containsString(network.ObjectMeta.Finalizers, networkFinalizer) {
			// Our finalizer is present, so let's handle any external dependency
			if err := r.deleteAnvilResources(ctx, &network); err != nil {
				return ctrl.Result{}, err
			}

			// Remove our finalizer from the list and update it
			network.ObjectMeta.Finalizers = removeString(network.ObjectMeta.Finalizers, networkFinalizer)
			if err := r.Update(ctx, &network); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
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

	// Check if the Anvil Pod already exists
	pod := &corev1.Pod{}
	err := r.Get(ctx, client.ObjectKey{Name: "anvil-pod", Namespace: network.Namespace}, pod)
	if err == nil {
		logger.Info("Anvil Pod already exists")
	} else if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "failed to get Anvil Pod")
		return err
	} else {
		// Define the Anvil Pod
		pod = &corev1.Pod{
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
						Command: []string{"anvil", "--chain-id", fmt.Sprintf("%d", network.Spec.ChainID), "--host", "0.0.0.0"},
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
	}

	// Check if the Anvil Service already exists
	service := &corev1.Service{}
	err = r.Get(ctx, client.ObjectKey{Name: "anvil-service", Namespace: network.Namespace}, service)
	if err == nil {
		logger.Info("Anvil Service already exists")
	} else if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "failed to get Anvil Service")
		return err
	} else {
		// Define the Anvil Service
		service = &corev1.Service{
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

	// Check if the Anvil RPC Secret already exists
	secret := &corev1.Secret{}
	err := r.Get(ctx, client.ObjectKey{Name: "anvil-rpc-secret", Namespace: network.Namespace}, secret)
	if err == nil {
		logger.Info("Anvil RPC Secret already exists")
	} else if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "failed to get Anvil RPC Secret")
		return err
	} else {
		// Define the Secret
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "anvil-rpc-secret",
				Namespace: network.Namespace,
			},
			StringData: map[string]string{
				"urlKey": fmt.Sprintf("http://anvil-service.%s.svc.cluster.local:8545", network.Namespace),
			},
		}

		// Create the Secret
		if err := r.Create(ctx, secret); err != nil {
			logger.Error(err, "failed to create Anvil RPC Secret")
			return err
		}
	}

	// Check if the Anvil RPCProvider already exists
	rpcProvider := &kontractdeployerv1alpha1.RPCProvider{}
	err = r.Get(ctx, client.ObjectKey{Name: "anvil", Namespace: network.Namespace}, rpcProvider)
	if err == nil {
		logger.Info("Anvil RPCProvider already exists")
	} else if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "failed to get Anvil RPCProvider")
		return err
	} else {
		// Define the RPCProvider
		rpcProvider = &kontractdeployerv1alpha1.RPCProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "anvil",
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
	}

	return nil
}

// createAnvilWallet creates a Wallet and Secret for the Anvil network
func (r *NetworkReconciler) createAnvilWallet(ctx context.Context, network *kontractdeployerv1alpha1.Network) error {
	logger := log.FromContext(ctx)

	// Check if the Anvil Wallet Secret already exists
	secret := &corev1.Secret{}
	err := r.Get(ctx, client.ObjectKey{Name: "anvil-wallet-secret", Namespace: network.Namespace}, secret)
	if err == nil {
		logger.Info("Anvil Wallet Secret already exists")
	} else if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "failed to get Anvil Wallet Secret")
		return err
	} else {
		// Define the Secret for the Anvil wallet
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "anvil-wallet-secret",
				Namespace: network.Namespace,
			},
			StringData: map[string]string{
				"privateKey": "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
				"publicKey":  "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
			},
		}

		// Create the Secret
		if err := r.Create(ctx, secret); err != nil {
			logger.Error(err, "failed to create Anvil Wallet Secret")
			return err
		}
	}

	// Check if the Anvil Wallet already exists
	wallet := &kontractdeployerv1alpha1.Wallet{}
	err = r.Get(ctx, client.ObjectKey{Name: "anvil-wallet", Namespace: network.Namespace}, wallet)
	if err == nil {
		logger.Info("Anvil Wallet already exists")
	} else if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "failed to get Anvil Wallet")
		return err
	} else {
		// Define the Wallet
		wallet = &kontractdeployerv1alpha1.Wallet{
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
				PublicKey: "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
				SecretRef: "anvil-wallet-secret",
			},
		}

		// Create the Wallet
		if err := r.Create(ctx, wallet); err != nil {
			logger.Error(err, "failed to create Anvil Wallet")
			return err
		}
	}

	return nil
}

// deleteAnvilResources deletes all resources related to the Anvil network
func (r *NetworkReconciler) deleteAnvilResources(ctx context.Context, network *kontractdeployerv1alpha1.Network) error {
	logger := log.FromContext(ctx)

	// Delete the Anvil Pod
	pod := &corev1.Pod{}
	err := r.Get(ctx, client.ObjectKey{Name: "anvil-pod", Namespace: network.Namespace}, pod)
	if err == nil {
		if err := r.Delete(ctx, pod); err != nil {
			logger.Error(err, "failed to delete Anvil Pod")
			return err
		}
	} else if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "failed to get Anvil Pod")
		return err
	}

	// Delete the Anvil Service
	service := &corev1.Service{}
	err = r.Get(ctx, client.ObjectKey{Name: "anvil-service", Namespace: network.Namespace}, service)
	if err == nil {
		if err := r.Delete(ctx, service); err != nil {
			logger.Error(err, "failed to delete Anvil Service")
			return err
		}
	} else if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "failed to get Anvil Service")
		return err
	}

	// Delete the Anvil Wallet Secret
	secret := &corev1.Secret{}
	err = r.Get(ctx, client.ObjectKey{Name: "anvil-wallet-secret", Namespace: network.Namespace}, secret)
	if err == nil {
		if err := r.Delete(ctx, secret); err != nil {
			logger.Error(err, "failed to delete Anvil Wallet Secret")
			return err
		}
	} else if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "failed to get Anvil Wallet Secret")
		return err
	}

	// Delete the Anvil Wallet
	wallet := &kontractdeployerv1alpha1.Wallet{}
	err = r.Get(ctx, client.ObjectKey{Name: "anvil-wallet", Namespace: network.Namespace}, wallet)
	if err == nil {
		if err := r.Delete(ctx, wallet); err != nil {
			logger.Error(err, "failed to delete Anvil Wallet")
			return err
		}
	} else if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "failed to get Anvil Wallet")
		return err
	}

	// Delete the Anvil RPCProvider Secret
	rpcSecret := &corev1.Secret{}
	err = r.Get(ctx, client.ObjectKey{Name: "anvil-rpc-secret", Namespace: network.Namespace}, rpcSecret)
	if err == nil {
		if err := r.Delete(ctx, rpcSecret); err != nil {
			logger.Error(err, "failed to delete Anvil RPC Secret")
			return err
		}
	} else if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "failed to get Anvil RPC Secret")
		return err
	}

	// Delete the Anvil RPCProvider
	rpcProvider := &kontractdeployerv1alpha1.RPCProvider{}
	err = r.Get(ctx, client.ObjectKey{Name: "anvil", Namespace: network.Namespace}, rpcProvider)
	if err == nil {
		if err := r.Delete(ctx, rpcProvider); err != nil {
			logger.Error(err, "failed to delete Anvil RPCProvider")
			return err
		}
	} else if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "failed to get Anvil RPCProvider")
		return err
	}

	return nil
}

// Helper functions to manage finalizers
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) []string {
	var result []string
	for _, item := range slice {
		if item != s {
			result = append(result, item)
		}
	}
	return result
}

// SetupWithManager sets up the controller with the Manager.
func (r *NetworkReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.EventRecorder = mgr.GetEventRecorderFor("network-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&kontractdeployerv1alpha1.Network{}).
		Complete(r)
}
