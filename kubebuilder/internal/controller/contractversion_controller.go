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
	err := r.Get(ctx, req.NamespacedName, contractVersion)
	if err != nil {
		if errors.IsNotFound(err) {
			// ContractVersion not found, ignore it
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}
	logger.Info("Fetching the ContractVersion instance", "ContractVersion.Name", contractVersion.Name)

	// Fetch the Network instance
	network := &kontractdeployerv1alpha1.Network{}
	err = r.Get(ctx, types.NamespacedName{Name: contractVersion.Spec.NetworkRef, Namespace: req.Namespace}, network)
	if err != nil {
		logger.Error(err, "Failed to get Network")
		return ctrl.Result{}, err
	}
	logger.Info("Fetching the Network instance", "Network.Name", network.Name)

	// Fetch the RPCProvider referenced by the Network
	rpcProvider := &kontractdeployerv1alpha1.RPCProvider{}
	err = r.Get(ctx, types.NamespacedName{Name: network.Spec.RPCProviderRef.Name, Namespace: req.Namespace}, rpcProvider)
	if err != nil {
		logger.Error(err, "Failed to get RPCProvider")
		return ctrl.Result{}, err
	}
	logger.Info("Fetching the RPCProvider instance", "RPCProvider.Name", rpcProvider.Name)

	// Fetch the Secret referenced by the RPCProvider
	rpcProviderSecret := &corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{Name: rpcProvider.Spec.SecretRef.Name, Namespace: req.Namespace}, rpcProviderSecret)
	if err != nil {
		logger.Error(err, "Failed to get RPCProvider Secret")
		return ctrl.Result{}, err
	}
	logger.Info("Fetching the RPCProvider Secret", "Secret.Name", rpcProvider.Spec.SecretRef.Name)

	// Fetch the Wallet instance
	wallet := &kontractdeployerv1alpha1.Wallet{}
	err = r.Get(ctx, types.NamespacedName{Name: contractVersion.Spec.WalletRef, Namespace: req.Namespace}, wallet)
	if err != nil {
		logger.Error(err, "Failed to get Wallet")
		return ctrl.Result{}, err
	}
	logger.Info("Fetching the Wallet instance", "Wallet.Name", wallet.Name)

	// Fetch the Wallet Secret
	if wallet.Status.SecretRef == "" {
		logger.Error(fmt.Errorf("wallet secret reference is empty"), "Wallet secret reference is empty", "Wallet.Name", wallet.Name)
		return ctrl.Result{}, fmt.Errorf("wallet secret reference is empty")
	}

	walletSecret := &corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{Name: wallet.Status.SecretRef, Namespace: req.Namespace}, walletSecret)
	if err != nil {
		logger.Error(err, "Failed to get Wallet Secret")
		return ctrl.Result{}, err
	}
	logger.Info("Fetching the Wallet Secret", "WalletSecret.Name", wallet.Status.SecretRef)

	// Create a ConfigMap for the contract code and tests
	configMapName := fmt.Sprintf("%s-contract", contractVersion.Name)
	configMapData := map[string]string{
		"code": contractVersion.Spec.Code,
	}

	// Only add the test data if it exists
	if contractVersion.Spec.Test != "" {
		configMapData["tests"] = contractVersion.Spec.Test
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
	err = r.createOrUpdateConfigMap(ctx, configMap)
	if err != nil {
		logger.Error(err, "Failed to create or update ConfigMap", "ConfigMap.Name", configMapName)
		return ctrl.Result{}, err
	}
	logger.Info("ConfigMap created or updated", "ConfigMap.Name", configMapName)

	// Define the job that will deploy the contract
	contractFileName := fmt.Sprintf("%s.sol", contractVersion.Spec.ContractName)
	testFileName := fmt.Sprintf("%s.t.sol", contractVersion.Spec.ContractName)

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
	if contractVersion.Spec.Test != "" {
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

	// Fetch and mount ConfigMaps for LocalModules
	localModuleNames := []string{}
	for _, module := range contractVersion.Spec.LocalModules {
		configMapName := module.Name
		localModuleNames = append(localModuleNames, configMapName)
		configMap := &corev1.ConfigMap{}
		err = r.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: req.Namespace}, configMap)
		if err != nil {
			logger.Error(err, "Failed to get LocalModule ConfigMap", "ConfigMap.Name", configMapName)
			return ctrl.Result{}, err
		}
		logger.Info("Fetching the LocalModule ConfigMap", "ConfigMap.Name", configMapName)

		for key := range configMap.Data {
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      configMapName,
				MountPath: fmt.Sprintf("/home/foundryuser/expedio-kontract-deployer/src/%s/%s", configMapName, key),
				SubPath:   key,
			})
		}

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
	}

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
							Name:  "foundry",
							Image: "docker.io/expedio/foundry:latest",
							Env: []corev1.EnvVar{
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
									Name: "RPC_KEY",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: rpcProvider.Spec.SecretRef.Name,
											},
											Key: rpcProvider.Spec.SecretRef.TokenKey,
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

	// Convert InitParams to JSON if not empty
	if len(contractVersion.Spec.InitParams) > 0 {
		initParamsJSON, err := json.Marshal(contractVersion.Spec.InitParams)
		if err != nil {
			logger.Error(err, "Failed to marshal InitParams to JSON")
			return ctrl.Result{}, err
		}
		// Add InitParams JSON to the job environment variables
		job.Spec.Template.Spec.Containers[0].Env = append(job.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
			Name:  "INIT_PARAMS",
			Value: string(initParamsJSON),
		})
	}

	// Fetch the BlockExplorer referenced by the Network, if it exists
	var blockExplorer *kontractdeployerv1alpha1.BlockExplorer
	if network.Spec.BlockExplorerRef != nil {
		blockExplorer = &kontractdeployerv1alpha1.BlockExplorer{}
		err = r.Get(ctx, types.NamespacedName{Name: network.Spec.BlockExplorerRef.Name, Namespace: req.Namespace}, blockExplorer)
		if err != nil {
			logger.Error(err, "Failed to get BlockExplorer")
			return ctrl.Result{}, err
		}
		logger.Info("Fetching the BlockExplorer instance", "BlockExplorer.Name", blockExplorer.Name)
	}

	// Add BlockExplorer details to the job environment variables if it exists
	if blockExplorer != nil {
		job.Spec.Template.Spec.Containers[0].Env = append(job.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
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

	// Add Chain ID to the job environment variables
	job.Spec.Template.Spec.Containers[0].Env = append(job.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
		Name:  "CHAIN_ID",
		Value: fmt.Sprintf("%d", network.Spec.ChainID),
	})

	// Set ContractVersion instance as the owner and controller of the Job
	if err := controllerutil.SetControllerReference(contractVersion, job, r.Scheme); err != nil {
		logger.Error(err, "Failed to set owner reference for Job", "Job.Name", job.Name)
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
			return ctrl.Result{}, err
		}
		// Job created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		logger.Error(err, "Failed to get Job")
		return ctrl.Result{}, err
	}

	// Job already exists - check its status
	if found.Status.Succeeded > 0 {
		// Job succeeded, update the ContractVersion status
		if contractVersion.Status.DeploymentTime.IsZero() {
			contractVersion.Status.DeploymentTime = metav1.Now()
		}
		contractVersion.Status.State = "deployed"
		if err := r.Status().Update(ctx, contractVersion); err != nil {
			logger.Error(err, "Failed to update ContractVersion status")
			return ctrl.Result{}, err
		}
	} else if found.Status.Failed > 0 {
		// Job failed, update the ContractVersion status
		contractVersion.Status.State = "failed"
		if err := r.Status().Update(ctx, contractVersion); err != nil {
			logger.Error(err, "Failed to update ContractVersion status")
			return ctrl.Result{}, err
		}
	}

	// Job is still running or pending - requeue
	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

// createOrUpdateConfigMap creates or updates a ConfigMap
func (r *ContractVersionReconciler) createOrUpdateConfigMap(ctx context.Context, cm *corev1.ConfigMap) error {
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
func (r *ContractVersionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kontractdeployerv1alpha1.ContractVersion{}).
		Complete(r)
}
