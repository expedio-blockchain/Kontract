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

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
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
	Scheme *runtime.Scheme
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

	// Create a ConfigMap for the contract code and tests
	configMapName := fmt.Sprintf("%s-contract", contract.Name)

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: req.Namespace,
		},
		Data: map[string]string{
			"code":  contract.Spec.Code,
			"tests": contract.Spec.Test,
		},
	}

	// Set Contract instance as the owner and controller of the ConfigMap
	if err := controllerutil.SetControllerReference(contract, configMap, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	// Create or update the ConfigMap
	err = r.createOrUpdateConfigMap(ctx, configMap)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Define the job that will deploy the contract
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
							Command: []string{
								"sh", "-c",
								fmt.Sprintf("forge create /contracts/%s.sol", contract.Spec.ContractName),
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "contract-code",
									MountPath: "/contracts",
									SubPath:   "code",
								},
								{
									Name:      "contract-tests",
									MountPath: "/tests",
									SubPath:   "tests",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
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
						{
							Name: "contract-tests",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: configMapName,
									},
								},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyOnFailure,
				},
			},
		},
	}

	// Set Contract instance as the owner and controller of the Job
	if err := controllerutil.SetControllerReference(contract, job, r.Scheme); err != nil {
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
			return ctrl.Result{}, err
		}
		// Job created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
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
