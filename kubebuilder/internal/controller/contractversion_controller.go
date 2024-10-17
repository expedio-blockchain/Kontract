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
	"reflect"
	"strings"
	"time"

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

// ContractVersionReconciler reconciles a ContractVersion object
type ContractVersionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=kontractdeployer.expedio.xyz,resources=contractversions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kontractdeployer.expedio.xyz,resources=contractversions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kontractdeployer.expedio.xyz,resources=contractversions/finalizers,verbs=update
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;create;update;delete

func (r *ContractVersionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the ContractVersion instance
	contractVersion := &kontractdeployerv1alpha1.ContractVersion{}
	if err := r.Get(ctx, req.NamespacedName, contractVersion); err != nil {
		if errors.IsNotFound(err) {
			// ContractVersion not found, ignore
			return ctrl.Result{}, nil
		}
		// Error reading the object
		return ctrl.Result{}, err
	}

	// Fetch the Network instance
	network := &kontractdeployerv1alpha1.Network{}
	if err := r.Get(ctx, types.NamespacedName{Name: contractVersion.Spec.NetworkRef, Namespace: req.Namespace}, network); err != nil {
		logger.Error(err, "Failed to get Network")
		return ctrl.Result{}, err
	}

	// Fetch the RPCProvider referenced by the Network
	rpcProvider := &kontractdeployerv1alpha1.RPCProvider{}
	if err := r.Get(ctx, types.NamespacedName{Name: network.Spec.RPCProviderRef.Name, Namespace: req.Namespace}, rpcProvider); err != nil {
		logger.Error(err, "Failed to get RPCProvider")
		return ctrl.Result{}, err
	}

	// Fetch the Secret referenced by the RPCProvider
	rpcProviderSecret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: rpcProvider.Spec.SecretRef.Name, Namespace: req.Namespace}, rpcProviderSecret); err != nil {
		logger.Error(err, "Failed to get RPCProvider Secret")
		return ctrl.Result{}, err
	}

	// Fetch the Wallet instance
	wallet := &kontractdeployerv1alpha1.Wallet{}
	if err := r.Get(ctx, types.NamespacedName{Name: contractVersion.Spec.WalletRef, Namespace: req.Namespace}, wallet); err != nil {
		logger.Error(err, "Failed to get Wallet")
		return ctrl.Result{}, err
	}

	// Fetch the Wallet Secret
	if wallet.Status.SecretRef == "" {
		err := fmt.Errorf("wallet secret reference is empty")
		logger.Error(err, "Wallet secret reference is empty", "Wallet.Name", wallet.Name)
		return ctrl.Result{}, err
	}

	walletSecret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: wallet.Status.SecretRef, Namespace: req.Namespace}, walletSecret); err != nil {
		logger.Error(err, "Failed to get Wallet Secret")
		return ctrl.Result{}, err
	}

	// Create a ConfigMap for the contract code, tests, script, and foundry.toml
	configMapName := fmt.Sprintf("%s-contract", contractVersion.Name)
	configMapData := map[string]string{
		"code": contractVersion.Spec.Code,
	}

	// Include foundry.toml if specified directly
	if contractVersion.Spec.FoundryConfig != "" {
		configMapData["foundry.toml"] = contractVersion.Spec.FoundryConfig
	}

	// Include test data if it exists
	if contractVersion.Spec.Test != "" {
		configMapData["tests"] = contractVersion.Spec.Test
	}

	// Include script data if it exists
	if contractVersion.Spec.Script != "" {
		configMapData["script"] = contractVersion.Spec.Script
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: req.Namespace,
		},
		Data: configMapData,
	}

	// Set ContractVersion instance as the owner and controller of the ConfigMap
	if err := controllerutil.SetControllerReference(contractVersion, configMap, r.Scheme); err != nil {
		logger.Error(err, "Failed to set owner reference for ConfigMap", "ConfigMap.Name", configMapName)
		return ctrl.Result{}, err
	}

	// Create or update the ConfigMap
	if err := r.createOrUpdateConfigMap(ctx, configMap); err != nil {
		logger.Error(err, "Failed to create or update ConfigMap", "ConfigMap.Name", configMapName)
		return ctrl.Result{}, err
	}

	// Prepare filenames for contract, test, and script
	contractFileName := fmt.Sprintf("%s.sol", contractVersion.Spec.ContractName)
	testFileName := fmt.Sprintf("%s.t.sol", contractVersion.Spec.ContractName)

	// Initialize volumes and volumeMounts
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

	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "contract-code",
			MountPath: fmt.Sprintf("/home/foundryuser/expedio-kontract-deployer/src/%s", contractFileName),
			SubPath:   "code",
		},
	}

	// Mount test data if it exists
	if contractVersion.Spec.Test != "" {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "contract-code",
			MountPath: fmt.Sprintf("/home/foundryuser/expedio-kontract-deployer/test/%s", testFileName),
			SubPath:   "tests",
		})
	}

	// Mount script data if it exists
	if contractVersion.Spec.Script != "" {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "contract-code",
			MountPath: "/home/foundryuser/expedio-kontract-deployer/script/script.s.sol",
			SubPath:   "script",
		})
	}

	// Add the test volume if the test data exists
	if contractVersion.Spec.Test != "" {
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

	// Add the script volume if the script data exists
	if contractVersion.Spec.Script != "" {
		volumes = append(volumes, corev1.Volume{
			Name: "contract-script",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: configMapName,
					},
				},
			},
		})
	}

	// Add foundry.toml mounting logic
	if contractVersion.Spec.FoundryConfigRef != nil {
		// FoundryConfigRef is specified, fetch the ConfigMap and mount it
		foundryConfigMapName := contractVersion.Spec.FoundryConfigRef.Name
		volumes = append(volumes, corev1.Volume{
			Name: "foundry-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: foundryConfigMapName,
					},
					Items: []corev1.KeyToPath{
						{
							Key:  "foundry.toml",
							Path: "foundry.toml",
						},
					},
				},
			},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "foundry-config",
			MountPath: "/home/foundryuser/expedio-kontract-deployer/foundry.toml",
			SubPath:   "foundry.toml",
		})
	} else if contractVersion.Spec.FoundryConfig != "" {
		// foundry.toml is included in the main ConfigMap
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "contract-code",
			MountPath: "/home/foundryuser/expedio-kontract-deployer/foundry.toml",
			SubPath:   "foundry.toml",
		})
	}

	// Fetch and mount ConfigMaps for LocalModules
	localModuleNames := []string{}
	for _, module := range contractVersion.Spec.LocalModules {
		configMapName := module.Name
		localModuleNames = append(localModuleNames, configMapName)
		configMap := &corev1.ConfigMap{}
		err := r.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: req.Namespace}, configMap)
		if err != nil {
			logger.Error(err, "Failed to get LocalModule ConfigMap", "ConfigMap.Name", configMapName)
			return ctrl.Result{}, err
		}

		// Add a volume for the module
		volumes = append(volumes, corev1.Volume{
			Name: configMapName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: configMapName,
					},
				},
			},
		})

		// Mount all keys in the ConfigMap
		for key := range configMap.Data {
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      configMapName,
				MountPath: fmt.Sprintf("/home/foundryuser/expedio-kontract-deployer/src/%s/%s", configMapName, key),
				SubPath:   key,
			})
		}
	}

	// Define environment variables for the job
	envVars := []corev1.EnvVar{
		{
			Name: "RPC_URL",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: rpcProvider.Spec.SecretRef.Name,
					},
					Key: rpcProvider.Spec.SecretRef.URLKey,
				},
			},
		},
		{
			Name: "WALLET_PRV_KEY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: wallet.Status.SecretRef,
					},
					Key: "privateKey",
				},
			},
		},
		{
			Name:  "CONTRACT_NAME",
			Value: contractVersion.Spec.ContractName,
		},
		{
			Name:  "EXTERNAL_MODULES",
			Value: strings.Join(contractVersion.Spec.ExternalModules, " "),
		},
		{
			Name:  "LOCAL_MODULES",
			Value: strings.Join(localModuleNames, " "),
		},
		{
			Name:  "CHAIN_ID",
			Value: fmt.Sprintf("%d", network.Spec.ChainID),
		},
	}

	// Add RPC_KEY if specified
	if rpcProvider.Spec.SecretRef.TokenKey != "" {
		envVars = append(envVars, corev1.EnvVar{
			Name: "RPC_KEY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: rpcProvider.Spec.SecretRef.Name,
					},
					Key: rpcProvider.Spec.SecretRef.TokenKey,
				},
			},
		})
	}

	// Fetch the BlockExplorer if referenced by the Network
	var blockExplorer *kontractdeployerv1alpha1.BlockExplorer
	if network.Spec.BlockExplorerRef != nil {
		blockExplorer = &kontractdeployerv1alpha1.BlockExplorer{}
		if err := r.Get(ctx, types.NamespacedName{Name: network.Spec.BlockExplorerRef.Name, Namespace: req.Namespace}, blockExplorer); err != nil {
			logger.Error(err, "Failed to get BlockExplorer")
			return ctrl.Result{}, err
		}
	}

	// Add BlockExplorer details to the environment variables if it exists
	if blockExplorer != nil {
		envVars = append(envVars, corev1.EnvVar{
			Name: "ETHERSCAN_API_URL",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: blockExplorer.Spec.SecretRef.Name,
					},
					Key: blockExplorer.Spec.SecretRef.URLKey,
				},
			},
		}, corev1.EnvVar{
			Name: "ETHERSCAN_API_KEY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: blockExplorer.Spec.SecretRef.Name,
					},
					Key: blockExplorer.Spec.SecretRef.TokenKey,
				},
			},
		})
	}

	// Convert InitParams to JSON if not empty
	if len(contractVersion.Spec.InitParams) > 0 {
		initParamsJSON, err := json.Marshal(contractVersion.Spec.InitParams)
		if err != nil {
			logger.Error(err, "Failed to marshal InitParams to JSON")
			return ctrl.Result{}, err
		}
		// Add InitParams JSON to the environment variables
		envVars = append(envVars, corev1.EnvVar{
			Name:  "INIT_PARAMS",
			Value: string(initParamsJSON),
		})
	}

	// Define the Job that will deploy the contract
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("contract-deploy-%s", contractVersion.Name),
			Namespace: req.Namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:         "foundry",
							Image:        "docker.io/expedio/foundry:latest",
							Env:          envVars,
							VolumeMounts: volumeMounts,
						},
					},
					Volumes:       volumes,
					RestartPolicy: corev1.RestartPolicyOnFailure,
				},
			},
		},
	}

	// Set ContractVersion instance as the owner and controller of the Job
	if err := controllerutil.SetControllerReference(contractVersion, job, r.Scheme); err != nil {
		logger.Error(err, "Failed to set owner reference for Job", "Job.Name", job.Name)
		return ctrl.Result{}, err
	}

	// Check if the Job already exists
	foundJob := &batchv1.Job{}
	if err := r.Get(ctx, client.ObjectKey{Name: job.Name, Namespace: job.Namespace}, foundJob); err != nil {
		if errors.IsNotFound(err) {
			// Job not found, create it
			logger.Info("Creating a new Job", "Job.Namespace", job.Namespace, "Job.Name", job.Name)
			if err := r.Create(ctx, job); err != nil {
				logger.Error(err, "Failed to create Job")
				return ctrl.Result{}, err
			}
			// Job created successfully - requeue
			return ctrl.Result{Requeue: true}, nil
		}
		// Error getting the Job
		logger.Error(err, "Failed to get Job")
		return ctrl.Result{}, err
	}

	// Job already exists - check its status
	if foundJob.Status.Succeeded > 0 {
		// Job succeeded, update the ContractVersion status
		if contractVersion.Status.DeploymentTime.IsZero() {
			contractVersion.Status.DeploymentTime = metav1.Now()
		}
		contractVersion.Status.State = "deployed"
		if err := r.Status().Update(ctx, contractVersion); err != nil {
			logger.Error(err, "Failed to update ContractVersion status")
			return ctrl.Result{}, err
		}
	} else if foundJob.Status.Failed > 0 {
		// Job failed, update the ContractVersion status
		contractVersion.Status.State = "failed"
		if err := r.Status().Update(ctx, contractVersion); err != nil {
			logger.Error(err, "Failed to update ContractVersion status")
			return ctrl.Result{}, err
		}
	} else {
		// Job is still running or pending - requeue
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	return ctrl.Result{}, nil
}

// createOrUpdateConfigMap creates or updates a ConfigMap
func (r *ContractVersionReconciler) createOrUpdateConfigMap(ctx context.Context, cm *corev1.ConfigMap) error {
	logger := log.FromContext(ctx)
	found := &corev1.ConfigMap{}
	if err := r.Get(ctx, client.ObjectKey{Name: cm.Name, Namespace: cm.Namespace}, found); err != nil {
		if errors.IsNotFound(err) {
			// ConfigMap not found, create it
			logger.Info("Creating ConfigMap", "ConfigMap.Name", cm.Name)
			return r.Create(ctx, cm)
		}
		// Error getting the ConfigMap
		return err
	}

	// Check if the existing ConfigMap data is different from the new data
	if !reflect.DeepEqual(found.Data, cm.Data) {
		// ConfigMap found, update it
		found.Data = cm.Data
		logger.Info("Updating ConfigMap", "ConfigMap.Name", cm.Name)
		return r.Update(ctx, found)
	}

	// No update needed
	return nil
}

// SetupWithManager sets up the controller with the Manager
func (r *ContractVersionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kontractdeployerv1alpha1.ContractVersion{}).
		Complete(r)
}
