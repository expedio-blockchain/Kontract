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
	"net/http"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kontractdeployerv1alpha1 "github.com/expedio-blockchain/KontractDeployer/api/v1alpha1"
)

// BlockExplorerReconciler reconciles a BlockExplorer object
type BlockExplorerReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder // Event recorder for logging events
}

// +kubebuilder:rbac:groups=kontractdeployer.expedio.xyz,resources=blockexplorers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kontractdeployer.expedio.xyz,resources=blockexplorers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kontractdeployer.expedio.xyz,resources=blockexplorers/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=create;update;get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch;update

// Reconcile is part of the main Kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *BlockExplorerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the BlockExplorer instance
	var blockExplorer kontractdeployerv1alpha1.BlockExplorer
	if err := r.Get(ctx, req.NamespacedName, &blockExplorer); err != nil {
		log.Error(err, "unable to fetch BlockExplorer")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Perform reconciliation logic if needed
	// (You can leave this part empty or perform some reconciliation logic if required)

	return ctrl.Result{}, nil
}

// StartPeriodicHealthCheck starts a background goroutine that checks the health of all BlockExplorers every 5 minutes
func (r *BlockExplorerReconciler) StartPeriodicHealthCheck(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(5 * time.Minute) // Changed from 1 minute to 5 minutes
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r.checkAllBlockExplorers(ctx)
			}
		}
	}()
}

// checkAllBlockExplorers checks the health of all BlockExplorers and updates their statuses
func (r *BlockExplorerReconciler) checkAllBlockExplorers(ctx context.Context) {
	log := log.FromContext(ctx)

	// List all BlockExplorers
	var blockExplorers kontractdeployerv1alpha1.BlockExplorerList
	if err := r.List(ctx, &blockExplorers); err != nil {
		log.Error(err, "unable to list BlockExplorers")
		return
	}

	for _, blockExplorer := range blockExplorers.Items {
		log := log.WithValues("BlockExplorer", blockExplorer.Name, "Namespace", blockExplorer.Namespace)

		// Fetch the referenced secret
		var secret corev1.Secret
		secretName := types.NamespacedName{Namespace: blockExplorer.Namespace, Name: blockExplorer.Spec.SecretRef.Name}
		if err := r.Get(ctx, secretName, &secret); err != nil {
			log.Error(err, fmt.Sprintf("BlockExplorer (%s) - unable to fetch Secret", blockExplorer.Name), "Secret", secretName)
			r.updateStatus(ctx, &blockExplorer, false, "")
			continue
		}

		// Extract API token and URL from the secret using the specified keys
		token := string(secret.Data[blockExplorer.Spec.SecretRef.TokenKey])
		apiEndpoint := string(secret.Data[blockExplorer.Spec.SecretRef.URLKey])

		// Validate the existence of the token and URL
		if token == "" || apiEndpoint == "" {
			log.Error(fmt.Errorf("missing token or URL"), fmt.Sprintf("BlockExplorer (%s) - missing required data in Secret", blockExplorer.Name))
			r.updateStatus(ctx, &blockExplorer, false, "")
			continue
		}

		// Perform the health check
		if err := r.checkAPIHealth(ctx, apiEndpoint, token, blockExplorer.Name); err != nil {
			log.Error(err, fmt.Sprintf("BlockExplorer (%s) - API health check failed", blockExplorer.Name))
			r.updateStatus(ctx, &blockExplorer, false, "")
			r.Recorder.Event(&blockExplorer, corev1.EventTypeWarning, "APIHealthCheckFailed", "API health check failed")
			continue
		}

		// Update the status to healthy: true without creating an event
		r.updateStatus(ctx, &blockExplorer, true, apiEndpoint)
	}
}

// checkAPIHealth sends a GET request to check the health of the BlockExplorer by verifying access to the API endpoint.
func (r *BlockExplorerReconciler) checkAPIHealth(ctx context.Context, endpoint, token, blockExplorerName string) error {
	log := log.FromContext(ctx)
	client := http.Client{
		Timeout: 10 * time.Second,
	}

	// Construct the URL for the health check
	url := fmt.Sprintf("%s?module=block&action=getblockreward&blockno=2165403&apikey=%s", strings.TrimRight(endpoint, "/"), token)
	log.Info(fmt.Sprintf("BlockExplorer (%s) - Performing API health check", blockExplorerName))

	// Perform the GET request
	resp, err := client.Get(url)
	if err != nil {
		log.Error(err, fmt.Sprintf("BlockExplorer (%s) - Failed to send API health check request", blockExplorerName))
		return err
	}
	defer resp.Body.Close()

	log.Info(fmt.Sprintf("BlockExplorer (%s) - Received response from API health check", blockExplorerName), "statusCode", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}

	// Decode the JSON response to verify it's a valid response
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Error(err, fmt.Sprintf("BlockExplorer (%s) - Failed to decode API health check response", blockExplorerName))
		return err
	}

	log.Info(fmt.Sprintf("BlockExplorer (%s) - Decoded API health check response", blockExplorerName))

	// Check if the status is "1" and message is "OK"
	if status, ok := result["status"].(string); !ok || status != "1" {
		return fmt.Errorf("invalid status in response: %v", result)
	}

	if message, ok := result["message"].(string); !ok || strings.ToUpper(message) != "OK" {
		return fmt.Errorf("invalid message in response: %v", result)
	}

	log.Info(fmt.Sprintf("BlockExplorer (%s) - API health check succeeded", blockExplorerName))
	return nil
}

// updateStatus updates the status of the BlockExplorer resource
func (r *BlockExplorerReconciler) updateStatus(ctx context.Context, blockExplorer *kontractdeployerv1alpha1.BlockExplorer, healthy bool, apiEndpoint string) {
	blockExplorer.Status.Healthy = healthy
	blockExplorer.Status.APIEndpoint = apiEndpoint
	if err := r.Status().Update(ctx, blockExplorer); err != nil {
		log.FromContext(ctx).Error(err, fmt.Sprintf("BlockExplorer (%s) - unable to update BlockExplorer status", blockExplorer.Name))
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *BlockExplorerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("blockexplorer-controller") // Initialize the event recorder
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&kontractdeployerv1alpha1.BlockExplorer{}).
		Complete(r); err != nil {
		return err
	}
	// Start the periodic health check
	r.StartPeriodicHealthCheck(context.Background())
	return nil
}
