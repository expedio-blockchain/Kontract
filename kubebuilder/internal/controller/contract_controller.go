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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	batchv1 "k8s.io/api/batch/v1"
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
	logger.Info("Fetching the Contract instance", "Contract.Name", contract.Name)

	// Fetch the Network instance
	network := &kontractdeployerv1alpha1.Network{}
	err = r.Get(ctx, types.NamespacedName{Name: contract.Spec.NetworkRef, Namespace: req.Namespace}, network)
	if err != nil {
		logger.Error(err, "Failed to get Network")
		r.Recorder.Error(err, "Failed to get Network", "NetworkRef", contract.Spec.NetworkRef)
		return ctrl.Result{}, err
	}
	logger.Info("Fetching the Network instance", "Network.Name", network.Name)

	// Fetch the RPCProvider referenced by the Network
	rpcProvider := &kontractdeployerv1alpha1.RPCProvider{}
	err = r.Get(ctx, types.NamespacedName{Name: network.Spec.RPCProviderRef.Name, Namespace: req.Namespace}, rpcProvider)
	if err != nil {
		logger.Error(err, "Failed to get RPCProvider")
		r.Recorder.Error(err, "Failed to get RPCProvider", "RPCProviderRef", network.Spec.RPCProviderRef.Name)
		return ctrl.Result{}, err
	}
	logger.Info("Fetching the RPCProvider instance", "RPCProvider.Name", rpcProvider.Name)

	// Fetch the Secret referenced by the RPCProvider
	rpcProviderSecret := &corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{Name: rpcProvider.Spec.SecretRef.Name, Namespace: req.Namespace}, rpcProviderSecret)
	if err != nil {
		logger.Error(err, "Failed to get RPCProvider Secret")
		r.Recorder.Error(err, "Failed to get RPCProvider Secret", "Secret.Name", rpcProvider.Spec.SecretRef.Name)
		return ctrl.Result{}, err
	}
	logger.Info("Fetching the RPCProvider Secret", "Secret.Name", rpcProvider.Spec.SecretRef.Name)

	// Fetch the Wallet instance
	wallet := &kontractdeployerv1alpha1.Wallet{}
	err = r.Get(ctx, types.NamespacedName{Name: contract.Spec.WalletRef, Namespace: req.Namespace}, wallet)
	if err != nil {
		logger.Error(err, "Failed to get Wallet")
		r.Recorder.Error(err, "Failed to get Wallet", "WalletRef", contract.Spec.WalletRef)
		return ctrl.Result{}, err
	}
	logger.Info("Fetching the Wallet instance", "Wallet.Name", wallet.Name)

	// Fetch the Wallet Secret
	walletSecret := &corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{Name: wallet.Spec.ImportFrom.SecretRef, Namespace: req.Namespace}, walletSecret)
	if err != nil {
		logger.Error(err, "Failed to get Wallet Secret")
		r.Recorder.Error(err, "Failed to get Wallet Secret", "WalletSecret.Name", wallet.Spec.ImportFrom.SecretRef)
		return ctrl.Result{}, err
	}
	logger.Info("Fetching the Wallet Secret", "WalletSecret.Name", wallet.Spec.ImportFrom.SecretRef)

	// Create a ConfigMap for the contract code and tests
	configMapName := fmt.Sprintf("%s-contract", contract.Name)
	configMapData := map[string]string{
		"code": contract.Spec.Code,
	}

	// Only add the test data if it exists
	if contract.Spec.Test != "" {
		configMapData["tests"] = contract.Spec.Test
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: req.Namespace,
		},
		Data: configMapData,
	}

	// Set Contract instance as the owner and controller of the ConfigMap
	if err := controllerutil.SetControllerReference(contract, configMap, r.Scheme); err != nil {
		r.Recorder.Error(err, "Failed to set owner reference for ConfigMap", "ConfigMap.Name", configMapName)
		return ctrl.Result{}, err
	}

	// Create or update the ConfigMap
	err = r.createOrUpdateConfigMap(ctx, configMap)
	if err != nil {
		r.Recorder.Error(err, "Failed to create or update ConfigMap", "ConfigMap.Name", configMapName)
		return ctrl.Result{}, err
	}
	logger.Info("ConfigMap created or updated", "ConfigMap.Name", configMapName)

	// Define the job that will deploy the contract
	contractFileName := fmt.Sprintf("%s.sol", contract.Spec.ContractName)
	testFileName := fmt.Sprintf("%s.t.sol", contract.Spec.ContractName)

	// Define the job that will deploy the contract
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "contract-code",
			MountPath: fmt.Sprintf("/home/foundryuser/expedio-kontract-deployer/src/%s", contractFileName),
			SubPath:   "code",
		},
	}

	volumes := []corev1.Volume{
		{
			Name: "contract-code",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: configMapName,
					},
				},
			},
		},
	}

	// Only add the test volume if the test data exists
	if contract.Spec.Test != "" {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "contract-tests",
			MountPath: fmt.Sprintf("/home/foundryuser/expedio-kontract-deployer/test/%s", testFileName),
			SubPath:   "tests",
		})

		volumes = append(volumes, corev1.Volume{
			Name: "contract-tests",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: configMapName,
					},
				},
			},
		})
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("contract-deploy-%s", contract.Name),
			Namespace: req.Namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "foundry",
							Image: "docker.io/expedio/foundry:latest",
							Env: []corev1.EnvVar{
								{
									Name:  "RPC_URL",
									Value: string(rpcProviderSecret.Data[rpcProvider.Spec.SecretRef.URLKey]),
								},
								{
									Name:  "RPC_KEY",
									Value: string(rpcProviderSecret.Data[rpcProvider.Spec.SecretRef.TokenKey]),
								},
								{
									Name:  "WALLET_PRV_KEY",
									Value: string(walletSecret.Data["privateKey"]),
								},
								{
									Name:  "CONTRACT_NAME",
									Value: contract.Spec.ContractName,
								},
								{
									Name:  "EXTERNAL_MODULES",
									Value: strings.Join(contract.Spec.ExternalModules, " "),
								},
								{
									Name:  "LOCAL_MODULES",
									Value: strings.Join(contract.Spec.LocalModules, " "),
								},
							},
							VolumeMounts: volumeMounts,
						},
					},
					Volumes:       volumes,
					RestartPolicy: corev1.RestartPolicyOnFailure,
				},
			},
		},
	}

	// Convert InitParams to JSON
	initParamsJSON, err := json.Marshal(contract.Spec.InitParams)
	if err != nil {
		logger.Error(err, "Failed to marshal InitParams to JSON")
		return ctrl.Result{}, err
	}

	// Add InitParams JSON to the job environment variables
	job.Spec.Template.Spec.Containers[0].Env = append(job.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
		Name:  "INIT_PARAMS",
		Value: string(initParamsJSON),
	})

	// Set Contract instance as the owner and controller of the Job
	if err := controllerutil.SetControllerReference(contract, job, r.Scheme); err != nil {
		r.Recorder.Error(err, "Failed to set owner reference for Job", "Job.Name", job.Name)
		return ctrl.Result{}, err
	}

	// Check if this Job already exists
	found := &batchv1.Job{}
	err = r.Get(ctx, client.ObjectKey{Name: job.Name, Namespace: job.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// Job not found, create it
		logger.Info("Creating a new Job", "Job.Namespace", job.Namespace, "Job.Name", job.Name)
		err = r.Create(ctx, job)
		if err != nil {
			logger.Error(err, "Failed to create Job")
			r.Recorder.Error(err, "Failed to create Job", "Job.Name", job.Name)
			return ctrl.Result{}, err
		}
		// Job created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		logger.Error(err, "Failed to get Job")
		return ctrl.Result{}, err
	}

	// Job already exists - don't requeue
	logger.Info("Skip reconcile: Job already exists", "Job.Namespace", found.Namespace, "Job.Name", found.Name)
	return ctrl.Result{}, nil
}

// createOrUpdateConfigMap creates or updates a ConfigMap
func (r *ContractReconciler) createOrUpdateConfigMap(ctx context.Context, cm *corev1.ConfigMap) error {
	found := &corev1.ConfigMap{}
	err := r.Get(ctx, client.ObjectKey{Name: cm.Name, Namespace: cm.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// ConfigMap not found, create it
		return r.Create(ctx, cm)
	} else if err != nil {
		return err
	}
	// ConfigMap found, update it
	found.Data = cm.Data
	return r.Update(ctx, found)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ContractReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kontractdeployerv1alpha1.Contract{}).
		Complete(r)
}
