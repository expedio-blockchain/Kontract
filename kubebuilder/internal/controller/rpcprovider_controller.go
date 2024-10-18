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

	kontractdeployerv1alpha1 "github.com/expedio-blockchain/Kontract/api/v1alpha1"
)

// RPCProviderReconciler reconciles an RPCProvider object
type RPCProviderReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder // Event recorder for logging events
}

// +kubebuilder:rbac:groups=kontract.expedio.xyz,resources=rpcproviders,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kontract.expedio.xyz,resources=rpcproviders/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kontract.expedio.xyz,resources=rpcproviders/finalizers,verbs=update
// +kubebuilder:rbac:groups=kontract.expedio.xyz,resources=wallets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kontract.expedio.xyz,resources=wallets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kontract.expedio.xyz,resources=wallets/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=create;update;get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch;update
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

// Reconcile is part of the main Kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *RPCProviderReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the RPCProvider instance
	var rpcProvider kontractdeployerv1alpha1.RPCProvider
	if err := r.Get(ctx, req.NamespacedName, &rpcProvider); err != nil {
		log.Error(err, "unable to fetch RPCProvider")
		r.Recorder.Event(&rpcProvider, corev1.EventTypeWarning, "FetchFailed", "Unable to fetch RPCProvider")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("Successfully fetched RPCProvider", "RPCProvider.Name", rpcProvider.Name)

	// Perform reconciliation logic if needed
	// (You can leave this part empty or perform some reconciliation logic if required)

	return ctrl.Result{}, nil
}

// StartPeriodicHealthCheck starts a background goroutine that checks the health of all RPCProviders every minute
func (r *RPCProviderReconciler) StartPeriodicHealthCheck(ctx context.Context) {
	go func() {

		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r.checkAllRPCProviders(ctx)
			}
		}
	}()
}

// checkAllRPCProviders checks the health of all RPCProviders and updates their statuses
func (r *RPCProviderReconciler) checkAllRPCProviders(ctx context.Context) {
	log := log.FromContext(ctx)

	// List all RPCProviders
	var rpcProviders kontractdeployerv1alpha1.RPCProviderList
	if err := r.List(ctx, &rpcProviders); err != nil {
		log.Error(err, "unable to list RPCProviders")
		return
	}

	for _, rpcProvider := range rpcProviders.Items {
		log := log.WithValues("RPCProvider", rpcProvider.Name, "Namespace", rpcProvider.Namespace)

		// Fetch the referenced secret
		var secret corev1.Secret
		secretName := types.NamespacedName{Namespace: rpcProvider.Namespace, Name: rpcProvider.Spec.SecretRef.Name}
		if err := r.Get(ctx, secretName, &secret); err != nil {
			log.Error(err, fmt.Sprintf("RPCProvider (%s) - unable to fetch Secret", rpcProvider.Name), "Secret", secretName)
			r.updateStatus(ctx, &rpcProvider, false, "")
			r.Recorder.Event(&rpcProvider, corev1.EventTypeWarning, "SecretFetchFailed", "Unable to fetch Secret for RPCProvider")
			continue
		}

		// Extract tokenKey and urlKey from the secret
		urlKey := string(secret.Data[rpcProvider.Spec.SecretRef.URLKey])
		tokenKey := ""
		if rpcProvider.Spec.SecretRef.TokenKey != "" {
			tokenKey = string(secret.Data[rpcProvider.Spec.SecretRef.TokenKey])
		}

		// Validate the existence of urlKey
		if urlKey == "" {
			log.Error(fmt.Errorf("missing URL key"), fmt.Sprintf("RPCProvider (%s) - missing required data in Secret", rpcProvider.Name))
			r.updateStatus(ctx, &rpcProvider, false, "")
			continue
		}

		// Construct the URL for the health check without logging the API key
		url := strings.TrimRight(urlKey, "/")
		if tokenKey != "" {
			url = fmt.Sprintf("%s/%s", url, tokenKey)
		}
		log.Info(fmt.Sprintf("RPCProvider (%s) - Performing periodic API health check", rpcProvider.Name))

		// Perform the health check
		if err := r.checkAPIHealth(ctx, url, rpcProvider.Name); err != nil {
			log.Error(err, fmt.Sprintf("RPCProvider (%s) - API health check failed", rpcProvider.Name))
			r.updateStatus(ctx, &rpcProvider, false, urlKey)
			r.Recorder.Event(&rpcProvider, corev1.EventTypeWarning, "APIHealthCheckFailed", "API health check failed")
			continue
		}

		// Update the status to healthy: true without creating an event
		r.updateStatus(ctx, &rpcProvider, true, urlKey)
	}
}

// checkAPIHealth sends a POST request to check the health of the RPC provider by calling `eth_blockNumber`.
func (r *RPCProviderReconciler) checkAPIHealth(ctx context.Context, url, rpcProviderName string) error {
	log := log.FromContext(ctx)
	client := http.Client{
		Timeout: 10 * time.Second,
	}

	// Mock blockchain operation - call eth_blockNumber
	requestBody := strings.NewReader(`{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`)
	req, err := http.NewRequest("POST", url, requestBody)
	if err != nil {
		log.Error(err, fmt.Sprintf("RPCProvider (%s) - Failed to create API health check request", rpcProviderName))
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	log.Info(fmt.Sprintf("RPCProvider (%s) - Sending API health check request", rpcProviderName))
	resp, err := client.Do(req)
	if err != nil {
		log.Error(err, fmt.Sprintf("RPCProvider (%s) - Failed to send API health check request", rpcProviderName))
		return err
	}
	defer resp.Body.Close()

	log.Info(fmt.Sprintf("RPCProvider (%s) - Received response from API health check", rpcProviderName), "statusCode", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}

	// Decode the JSON response to verify it's a valid JSON-RPC response
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Error(err, fmt.Sprintf("RPCProvider (%s) - Failed to decode API health check response", rpcProviderName))
		return err
	}

	log.Info(fmt.Sprintf("RPCProvider (%s) - Decoded API health check response", rpcProviderName))

	if _, ok := result["result"]; !ok {
		return fmt.Errorf("invalid JSON-RPC response: %v", result)
	}

	log.Info(fmt.Sprintf("RPCProvider (%s) - API health check succeeded", rpcProviderName))
	return nil
}

// updateStatus updates the status of the RPCProvider resource
func (r *RPCProviderReconciler) updateStatus(ctx context.Context, rpcProvider *kontractdeployerv1alpha1.RPCProvider, healthy bool, apiEndpoint string) {
	rpcProvider.Status.Healthy = healthy
	rpcProvider.Status.APIEndpoint = apiEndpoint
	if err := r.Status().Update(ctx, rpcProvider); err != nil {
		log.FromContext(ctx).Error(err, fmt.Sprintf("RPCProvider (%s) - unable to update RPCProvider status", rpcProvider.Name))
		r.Recorder.Event(rpcProvider, corev1.EventTypeWarning, "StatusUpdateFailed", "Failed to update RPCProvider status")
	} else {
		log.FromContext(ctx).Info("RPCProvider status updated successfully", "RPCProvider.Name", rpcProvider.Name)
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *RPCProviderReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("rpcprovider-controller") // Initialize the event recorder
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&kontractdeployerv1alpha1.RPCProvider{}).
		Complete(r); err != nil {
		return err
	}
	// Start the periodic health check
	r.StartPeriodicHealthCheck(context.Background())
	return nil
}
