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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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

func (r *ContractReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the Contract instance
	contract := &kontractdeployerv1alpha1.Contract{}
	if err := r.Get(ctx, req.NamespacedName, contract); err != nil {
		if errors.IsNotFound(err) {
			// Contract not found, ignore it
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		logger.Error(err, "Failed to get Contract")
		return ctrl.Result{}, err
	}

	// Extract Code, Test, Script, and FoundryConfig from ConfigMaps if references are provided
	code, err := r.getConfigMapData(ctx, req.Namespace, contract.Spec.CodeRef)
	if err != nil {
		logger.Error(err, "Failed to get Code from ConfigMap")
		return ctrl.Result{}, err
	}

	test, err := r.getConfigMapData(ctx, req.Namespace, contract.Spec.TestRef)
	if err != nil {
		logger.Error(err, "Failed to get Test from ConfigMap")
		return ctrl.Result{}, err
	}

	script, err := r.getConfigMapData(ctx, req.Namespace, contract.Spec.ScriptRef)
	if err != nil {
		logger.Error(err, "Failed to get Script from ConfigMap")
		return ctrl.Result{}, err
	}

	foundryConfig, err := r.getConfigMapData(ctx, req.Namespace, contract.Spec.FoundryConfigRef)
	if err != nil {
		logger.Error(err, "Failed to get FoundryConfig from ConfigMap")
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
				Code:            code,
				Test:            test,
				InitParams:      contract.Spec.InitParams,
				ExternalModules: contract.Spec.ExternalModules,
				LocalModules:    contract.Spec.LocalModules,
				Script:          script,
				FoundryConfig:   foundryConfig,
			},
		}

		// Set Contract instance as the owner and controller of the ContractVersion
		if err := controllerutil.SetControllerReference(contract, contractVersion, r.Scheme); err != nil {
			logger.Error(err, "Failed to set owner reference for ContractVersion", "ContractVersion.Name", contractVersion.Name)
			return ctrl.Result{}, err
		}

		// Create the ContractVersion
		if err := r.Create(ctx, contractVersion); err != nil {
			if !errors.IsAlreadyExists(err) {
				logger.Error(err, "Failed to create ContractVersion", "ContractVersion.Name", contractVersion.Name)
				return ctrl.Result{}, err
			}
		}
	}

	// Update the Contract status with the current version
	contract.Status.CurrentVersion = fmt.Sprintf("%s-version-%d", contract.Name, contract.Generation)
	if err := r.Status().Update(ctx, contract); err != nil {
		logger.Error(err, "Failed to update Contract status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// getConfigMapData fetches the data from a ConfigMap based on the provided reference
func (r *ContractReconciler) getConfigMapData(ctx context.Context, namespace string, ref *kontractdeployerv1alpha1.ConfigMapKeyReference) (string, error) {
	if ref == nil {
		return "", nil
	}

	configMap := &corev1.ConfigMap{}
	if err := r.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: namespace}, configMap); err != nil {
		return "", err
	}

	data, exists := configMap.Data[ref.Key]
	if !exists {
		return "", fmt.Errorf("key %s not found in ConfigMap %s", ref.Key, ref.Name)
	}

	return data, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ContractReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kontractdeployerv1alpha1.Contract{}).
		Complete(r)
}
