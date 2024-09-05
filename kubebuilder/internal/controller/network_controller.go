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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kontractdeployerv1alpha1 "github.com/expedio-blockchain/KontractDeployer/api/v1alpha1"
)

// NetworkReconciler reconciles a Network object
type NetworkReconciler struct {
	client.Client
	Scheme *runtime.Scheme
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

	// Fetch the referenced RPCProvider
	var rpcProvider kontractdeployerv1alpha1.RPCProvider
	if err := r.Get(ctx, client.ObjectKey{Name: network.Spec.RPCProviderRef.Name, Namespace: network.Namespace}, &rpcProvider); err != nil {
		logger.Error(err, "unable to fetch RPCProvider")
		return ctrl.Result{}, err
	}

	// Fetch the referenced BlockExplorer if it exists
	if network.Spec.BlockExplorerRef != nil {
		var blockExplorer kontractdeployerv1alpha1.BlockExplorer
		if err := r.Get(ctx, client.ObjectKey{Name: network.Spec.BlockExplorerRef.Name, Namespace: network.Namespace}, &blockExplorer); err != nil {
			logger.Error(err, "unable to fetch BlockExplorer")
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

// SetupWithManager sets up the controller with the Manager.
func (r *NetworkReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kontractdeployerv1alpha1.Network{}).
		Complete(r)
}
