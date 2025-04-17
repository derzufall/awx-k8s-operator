/*
Copyright 2023.

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

package controllers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	awxv1alpha1 "github.com/derzufall/awx-k8s-operator/api/v1alpha1"
	"github.com/derzufall/awx-k8s-operator/pkg/awx"
)

// AWXInstanceReconciler reconciles a AWXInstance object
type AWXInstanceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=awx.ansible.com,resources=awxinstances,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=awx.ansible.com,resources=awxinstances/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=awx.ansible.com,resources=awxinstances/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.0/pkg/reconcile
func (r *AWXInstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the AWXInstance resource
	instance := &awxv1alpha1.AWXInstance{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request
		return ctrl.Result{}, err
	}

	// Initialize status maps if they don't exist
	if instance.Status.ProjectStatuses == nil {
		instance.Status.ProjectStatuses = make(map[string]string)
	}
	if instance.Status.InventoryStatuses == nil {
		instance.Status.InventoryStatuses = make(map[string]string)
	}
	if instance.Status.JobTemplateStatuses == nil {
		instance.Status.JobTemplateStatuses = make(map[string]string)
	}

	// Initialize or update the LastConnectionCheck timestamp if needed
	if instance.Status.LastConnectionCheck.IsZero() {
		instance.Status.LastConnectionCheck = metav1.Now()
		if err := r.Status().Update(ctx, instance); err != nil {
			logger.Error(err, "Failed to update LastConnectionCheck timestamp")
			return ctrl.Result{}, err
		}
	}

	// Define a finalizer to clean up AWX resources when the CR is deleted
	awxFinalizer := "awx.ansible.com/finalizer"

	// Check if the AWXInstance is being deleted
	if instance.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(instance, awxFinalizer) {
			// Run finalization logic
			if err := r.finalizeAWXInstance(ctx, instance); err != nil {
				return ctrl.Result{}, err
			}

			// Remove finalizer once cleanup is done
			controllerutil.RemoveFinalizer(instance, awxFinalizer)
			if err := r.Update(ctx, instance); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(instance, awxFinalizer) {
		controllerutil.AddFinalizer(instance, awxFinalizer)
		if err := r.Update(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Set the protocol, defaulting to https if not specified
	protocol := "https"
	if instance.Spec.Protocol != "" {
		protocol = instance.Spec.Protocol
	}

	// Create AWX client
	baseURL := fmt.Sprintf("%s://%s", protocol, instance.Spec.Hostname)
	awxClient := awx.NewClient(baseURL, instance.Spec.AdminUser, instance.Spec.AdminPassword)

	// Check if we need to perform a periodic connection test (every 30 seconds)
	now := metav1.Now()
	timeSinceLastCheck := now.Time.Sub(instance.Status.LastConnectionCheck.Time)
	if timeSinceLastCheck >= 30*time.Second {
		logger.Info("Performing periodic connection test",
			"instance", instance.Name,
			"hostname", instance.Spec.Hostname,
			"timeSinceLastCheck", timeSinceLastCheck.String())

		// Update the LastConnectionCheck timestamp
		instance.Status.LastConnectionCheck = now

		// Test connection to AWX
		connectionErr := r.testConnection(ctx, awxClient)
		if connectionErr != nil {
			// Update connection status
			instance.Status.ConnectionStatus = fmt.Sprintf("Failed: %v", connectionErr)
			logger.Error(connectionErr, "Periodic connection test failed",
				"instance", instance.Name,
				"hostname", instance.Spec.Hostname,
				"protocol", protocol,
				"user", instance.Spec.AdminUser)
		} else {
			// Connection successful
			instance.Status.ConnectionStatus = "Connected"
			logger.Info("Periodic connection test successful",
				"instance", instance.Name,
				"hostname", instance.Spec.Hostname)
		}

		// Update status with new connection information
		if err := r.Status().Update(ctx, instance); err != nil {
			logger.Error(err, "Failed to update connection status")
			return ctrl.Result{}, err
		}

		// If this is an external instance and connection failed, don't proceed with reconciliation
		if connectionErr != nil && instance.Spec.ExternalInstance {
			meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
				Type:               "Ready",
				Status:             metav1.ConditionFalse,
				LastTransitionTime: metav1.Now(),
				Reason:             "ConnectionFailed",
				Message:            fmt.Sprintf("Failed to connect to external AWX instance: %v", connectionErr),
			})

			if err := r.Status().Update(ctx, instance); err != nil {
				logger.Error(err, "Failed to update AWXInstance status")
			}

			return ctrl.Result{RequeueAfter: 30 * time.Second}, connectionErr
		}
	} else {
		// Test connection to AWX if we're not doing a periodic check
		if err := r.testConnection(ctx, awxClient); err != nil {
			logger.Error(err, "Failed to connect to AWX instance",
				"instance", instance.Name,
				"hostname", instance.Spec.Hostname,
				"protocol", protocol,
				"user", instance.Spec.AdminUser)

			// If this is an external instance, we expect it to exist
			if instance.Spec.ExternalInstance {
				meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
					Type:               "Ready",
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
					Reason:             "ConnectionFailed",
					Message:            fmt.Sprintf("Failed to connect to external AWX instance: %v", err),
				})

				if err := r.Status().Update(ctx, instance); err != nil {
					logger.Error(err, "Failed to update AWXInstance status")
				}

				return ctrl.Result{RequeueAfter: 30 * time.Second}, err
			}

			// For non-external instances, this may be expected during initial setup
			logger.Info("AWX instance not available yet, will retry")
		}
	}

	// Check and reconcile any differences from AWX internal state to the desired state
	if changed, err := r.reconcileInternalChanges(ctx, instance, awxClient); err != nil {
		logger.Error(err, "Failed to reconcile internal AWX changes",
			"instance", instance.Name,
			"details", err.Error())
		return ctrl.Result{RequeueAfter: time.Minute}, err
	} else if changed {
		logger.Info("Detected and corrected internal AWX changes", "instance", instance.Name)
		// If changes were detected and corrected, update the status
		if err := r.Status().Update(ctx, instance); err != nil {
			logger.Error(err, "Failed to update AWXInstance status")
			return ctrl.Result{}, err
		}
	}

	// Reconcile Projects
	projectManager := awx.NewProjectManager(awxClient)
	for _, projectSpec := range instance.Spec.Projects {
		logger.Info("Reconciling project", "name", projectSpec.Name, "instance", instance.Name)
		_, err := projectManager.EnsureProject(projectSpec)
		if err != nil {
			logger.Error(err, "Failed to reconcile project",
				"name", projectSpec.Name,
				"instance", instance.Name,
				"details", err.Error())
			instance.Status.ProjectStatuses[projectSpec.Name] = fmt.Sprintf("Failed: %v", err)

			// Update reconciliation status
			if err := r.Status().Update(ctx, instance); err != nil {
				logger.Error(err, "Failed to update AWXInstance status")
				return ctrl.Result{}, err
			}

			return ctrl.Result{RequeueAfter: time.Minute}, err
		}
		instance.Status.ProjectStatuses[projectSpec.Name] = "Reconciled"
	}

	// Reconcile Inventories
	inventoryManager := awx.NewInventoryManager(awxClient)
	for _, inventorySpec := range instance.Spec.Inventories {
		logger.Info("Reconciling inventory", "name", inventorySpec.Name, "instance", instance.Name)
		_, err := inventoryManager.EnsureInventory(inventorySpec)
		if err != nil {
			logger.Error(err, "Failed to reconcile inventory",
				"name", inventorySpec.Name,
				"instance", instance.Name,
				"details", err.Error())
			instance.Status.InventoryStatuses[inventorySpec.Name] = fmt.Sprintf("Failed: %v", err)

			// Update reconciliation status
			if err := r.Status().Update(ctx, instance); err != nil {
				logger.Error(err, "Failed to update AWXInstance status")
				return ctrl.Result{}, err
			}

			return ctrl.Result{RequeueAfter: time.Minute}, err
		}
		instance.Status.InventoryStatuses[inventorySpec.Name] = "Reconciled"
	}

	// Reconcile Job Templates (after projects and inventories)
	jobTemplateManager := awx.NewJobTemplateManager(awxClient)
	for _, jobTemplateSpec := range instance.Spec.JobTemplates {
		logger.Info("Reconciling job template", "name", jobTemplateSpec.Name, "instance", instance.Name)
		_, err := jobTemplateManager.EnsureJobTemplate(jobTemplateSpec)
		if err != nil {
			logger.Error(err, "Failed to reconcile job template",
				"name", jobTemplateSpec.Name,
				"instance", instance.Name,
				"details", err.Error())
			instance.Status.JobTemplateStatuses[jobTemplateSpec.Name] = fmt.Sprintf("Failed: %v", err)

			// Update reconciliation status
			if err := r.Status().Update(ctx, instance); err != nil {
				logger.Error(err, "Failed to update AWXInstance status")
				return ctrl.Result{}, err
			}

			return ctrl.Result{RequeueAfter: time.Minute}, err
		}
		instance.Status.JobTemplateStatuses[jobTemplateSpec.Name] = "Reconciled"
	}

	// Update Ready condition
	meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             "ReconciliationSucceeded",
		Message:            "AWXInstance resources have been reconciled successfully",
	})

	// Update status
	if err := r.Status().Update(ctx, instance); err != nil {
		logger.Error(err, "Failed to update AWXInstance status")
		return ctrl.Result{}, err
	}

	// Requeue after 30 seconds to ensure connection tests run regularly
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// reconcileInternalChanges checks if AWX's internal state matches the desired state
// and corrects any differences found. Returns true if changes were detected and corrected.
func (r *AWXInstanceReconciler) reconcileInternalChanges(ctx context.Context,
	instance *awxv1alpha1.AWXInstance, awxClient *awx.Client) (bool, error) {

	logger := log.FromContext(ctx)
	changesDetected := false

	// Create managers for each resource type
	projectManager := awx.NewProjectManager(awxClient)
	inventoryManager := awx.NewInventoryManager(awxClient)
	jobTemplateManager := awx.NewJobTemplateManager(awxClient)

	// Check Projects
	for _, projectSpec := range instance.Spec.Projects {
		logger.Info("Checking project state", "name", projectSpec.Name)
		project, err := projectManager.GetProject(projectSpec.Name)
		if err != nil {
			return false, fmt.Errorf("failed to get project %s: %w", projectSpec.Name, err)
		}

		// If project doesn't exist or its configuration doesn't match the spec, reconcile it
		if project == nil || !projectManager.IsProjectInDesiredState(project, projectSpec) {
			logger.Info("Project needs reconciliation", "name", projectSpec.Name)
			_, err := projectManager.EnsureProject(projectSpec)
			if err != nil {
				return false, fmt.Errorf("failed to reconcile project %s: %w", projectSpec.Name, err)
			}
			instance.Status.ProjectStatuses[projectSpec.Name] = "Reconciled (corrected internal changes)"
			changesDetected = true
		}
	}

	// Check Inventories
	for _, inventorySpec := range instance.Spec.Inventories {
		logger.Info("Checking inventory state", "name", inventorySpec.Name)
		inventory, err := inventoryManager.GetInventory(inventorySpec.Name)
		if err != nil {
			return false, fmt.Errorf("failed to get inventory %s: %w", inventorySpec.Name, err)
		}

		// If inventory doesn't exist or its configuration doesn't match the spec, reconcile it
		if inventory == nil || !inventoryManager.IsInventoryInDesiredState(inventory, inventorySpec) {
			logger.Info("Inventory needs reconciliation", "name", inventorySpec.Name)
			_, err := inventoryManager.EnsureInventory(inventorySpec)
			if err != nil {
				return false, fmt.Errorf("failed to reconcile inventory %s: %w", inventorySpec.Name, err)
			}
			instance.Status.InventoryStatuses[inventorySpec.Name] = "Reconciled (corrected internal changes)"
			changesDetected = true
		}
	}

	// Check Job Templates
	for _, jobTemplateSpec := range instance.Spec.JobTemplates {
		logger.Info("Checking job template state", "name", jobTemplateSpec.Name)
		jobTemplate, err := jobTemplateManager.GetJobTemplate(jobTemplateSpec.Name)
		if err != nil {
			return false, fmt.Errorf("failed to get job template %s: %w", jobTemplateSpec.Name, err)
		}

		// If job template doesn't exist or its configuration doesn't match the spec, reconcile it
		if jobTemplate == nil || !jobTemplateManager.IsJobTemplateInDesiredState(jobTemplate, jobTemplateSpec) {
			logger.Info("Job template needs reconciliation", "name", jobTemplateSpec.Name)
			_, err := jobTemplateManager.EnsureJobTemplate(jobTemplateSpec)
			if err != nil {
				return false, fmt.Errorf("failed to reconcile job template %s: %w", jobTemplateSpec.Name, err)
			}
			instance.Status.JobTemplateStatuses[jobTemplateSpec.Name] = "Reconciled (corrected internal changes)"
			changesDetected = true
		}
	}

	return changesDetected, nil
}

// finalizeAWXInstance performs cleanup when the instance is being deleted
func (r *AWXInstanceReconciler) finalizeAWXInstance(ctx context.Context, instance *awxv1alpha1.AWXInstance) error {
	logger := log.FromContext(ctx)
	logger.Info("Finalizing AWXInstance", "name", instance.Name)

	// Set the protocol, defaulting to https if not specified
	protocol := "https"
	if instance.Spec.Protocol != "" {
		protocol = instance.Spec.Protocol
	}

	// Create AWX client
	baseURL := fmt.Sprintf("%s://%s", protocol, instance.Spec.Hostname)
	awxClient := awx.NewClient(baseURL, instance.Spec.AdminUser, instance.Spec.AdminPassword)

	// Delete job templates first (as they depend on projects and inventories)
	jobTemplateManager := awx.NewJobTemplateManager(awxClient)
	for _, jobTemplateSpec := range instance.Spec.JobTemplates {
		logger.Info("Deleting job template", "name", jobTemplateSpec.Name)
		err := jobTemplateManager.DeleteJobTemplate(jobTemplateSpec.Name)
		if err != nil {
			logger.Error(err, "Failed to delete job template", "name", jobTemplateSpec.Name)
			return err
		}
	}

	// Delete inventories
	inventoryManager := awx.NewInventoryManager(awxClient)
	for _, inventorySpec := range instance.Spec.Inventories {
		logger.Info("Deleting inventory", "name", inventorySpec.Name)
		err := inventoryManager.DeleteInventory(inventorySpec.Name)
		if err != nil {
			logger.Error(err, "Failed to delete inventory", "name", inventorySpec.Name)
			return err
		}
	}

	// Delete projects
	projectManager := awx.NewProjectManager(awxClient)
	for _, projectSpec := range instance.Spec.Projects {
		logger.Info("Deleting project", "name", projectSpec.Name)
		err := projectManager.DeleteProject(projectSpec.Name)
		if err != nil {
			logger.Error(err, "Failed to delete project", "name", projectSpec.Name)
			return err
		}
	}

	logger.Info("Successfully finalized AWXInstance", "name", instance.Name)
	return nil
}

// testConnection tests connectivity to the AWX instance
func (r *AWXInstanceReconciler) testConnection(ctx context.Context, awxClient *awx.Client) error {
	logger := log.FromContext(ctx)
	logger.Info("Testing connection to AWX instance")

	// Use the client's TestConnection method
	err := awxClient.TestConnection()
	if err != nil {
		// Parse the error message to provide more context
		var errorDetails string
		if strings.Contains(err.Error(), "failed to connect") {
			errorDetails = "Network connectivity issue - check network routes and firewall rules"
		} else if strings.Contains(err.Error(), "unexpected status code: 401") {
			errorDetails = "Authentication failed - check username and password"
		} else if strings.Contains(err.Error(), "unexpected status code: 404") {
			errorDetails = "API endpoint not found - check AWX URL and API path"
		} else if strings.Contains(err.Error(), "context deadline exceeded") ||
			strings.Contains(err.Error(), "timeout") {
			errorDetails = "Connection timed out - check if AWX service is running and network latency"
		} else {
			errorDetails = fmt.Sprintf("Unknown error: %v", err)
		}

		logger.Error(err, "Failed to connect to AWX instance",
			"errorType", errorDetails)
		return fmt.Errorf("%s: %w", errorDetails, err)
	}

	logger.Info("Successfully connected to AWX instance")
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AWXInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&awxv1alpha1.AWXInstance{}).
		Complete(r)
}
