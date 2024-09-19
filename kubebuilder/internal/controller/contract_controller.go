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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kontractdeployerv1alpha1 "github.com/expedio-blockchain/KontractDeployer/api/v1alpha1"
)

// ContractReconciler reconciles a Contract object
type ContractReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder *logr.Logger
}

// +kubebuilder:rbac:groups=kontractdeployer.expedio.xyz,resources=contracts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kontractdeployer.expedio.xyz,resources=contracts/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kontractdeployer.expedio.xyz,resources=contracts/finalizers,verbs=update
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;create;update;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ContractReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the Contract instance
	contract := &kontractdeployerv1alpha1.Contract{}
	err := r.Get(ctx, req.NamespacedName, contract)
	if err != nil {
		if errors.IsNotFound(err) {
			// Contract not found, ignore it
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	// Iterate over each network reference and create a ContractVersion
	for _, networkRef := range contract.Spec.NetworkRefs {
		contractVersion := &kontractdeployerv1alpha1.ContractVersion{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s-version-%d", contract.Name, networkRef, contract.Generation),
				Namespace: req.Namespace,
			},
			Spec: kontractdeployerv1alpha1.ContractVersionSpec{
				ContractName:    contract.Spec.ContractName,
				NetworkRef:      networkRef,
				WalletRef:       contract.Spec.WalletRef,
				GasStrategyRef:  contract.Spec.GasStrategyRef,
				Code:            contract.Spec.Code,
				Test:            contract.Spec.Test,
				InitParams:      contract.Spec.InitParams,
				ExternalModules: contract.Spec.ExternalModules,
				LocalModules:    contract.Spec.LocalModules,
				Script:          contract.Spec.Script,
			},
		}

		// Set Contract instance as the owner and controller of the ContractVersion
		if err := controllerutil.SetControllerReference(contract, contractVersion, r.Scheme); err != nil {
			r.Recorder.Error(err, "Failed to set owner reference for ContractVersion", "ContractVersion.Name", contractVersion.Name)
			return ctrl.Result{}, err
		}

		// Create the ContractVersion
		err = r.Create(ctx, contractVersion)
		if err != nil {
			if !errors.IsAlreadyExists(err) {
				logger.Error(err, "Failed to create ContractVersion")
				r.Recorder.Error(err, "Failed to create ContractVersion", "ContractVersion.Name", contractVersion.Name)
				return ctrl.Result{}, err
			}
		}
	}

	// Update the Contract status with the current version
	contract.Status.CurrentVersion = fmt.Sprintf("%s-version-%d", contract.Name, contract.Generation)
	err = r.Status().Update(ctx, contract)
	if err != nil {
		logger.Error(err, "Failed to update Contract status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ContractReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kontractdeployerv1alpha1.Contract{}).
		Complete(r)
}
