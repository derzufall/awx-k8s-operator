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
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	awxv1alpha1 "github.com/yourusername/awx-operator/api/v1alpha1"
	"github.com/yourusername/awx-operator/pkg/awx"
)

// AWXInstanceReconciler reconciles a AWXInstance object
type AWXInstanceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=awx.example.com,resources=awxinstances,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=awx.example.com,resources=awxinstances/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=awx.example.com,resources=awxinstances/finalizers,verbs=update

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

	// Define a finalizer to clean up AWX resources when the CR is deleted
	awxFinalizer := "awx.example.com/finalizer"

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

	// Create AWX client
	// Protocol, hostname and port should be customized based on your setup
	baseURL := fmt.Sprintf("https://%s", instance.Spec.Hostname)
	awxClient := awx.NewClient(baseURL, instance.Spec.AdminUser, instance.Spec.AdminPassword)

	// Check and reconcile any differences from AWX internal state to the desired state
	if changed, err := r.reconcileInternalChanges(ctx, instance, awxClient); err != nil {
		logger.Error(err, "Failed to reconcile internal AWX changes")
		return ctrl.Result{RequeueAfter: time.Minute}, err
	} else if changed {
		logger.Info("Detected and corrected internal AWX changes")
		// If changes were detected and corrected, update the status
		if err := r.Status().Update(ctx, instance); err != nil {
			logger.Error(err, "Failed to update AWXInstance status")
			return ctrl.Result{}, err
		}
	}

	// Reconcile Projects
	projectManager := awx.NewProjectManager(awxClient)
	for _, projectSpec := range instance.Spec.Projects {
		logger.Info("Reconciling project", "name", projectSpec.Name)
		_, err := projectManager.EnsureProject(projectSpec)
		if err != nil {
			logger.Error(err, "Failed to reconcile project", "name", projectSpec.Name)
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
		logger.Info("Reconciling inventory", "name", inventorySpec.Name)
		_, err := inventoryManager.EnsureInventory(inventorySpec)
		if err != nil {
			logger.Error(err, "Failed to reconcile inventory", "name", inventorySpec.Name)
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
		logger.Info("Reconciling job template", "name", jobTemplateSpec.Name)
		_, err := jobTemplateManager.EnsureJobTemplate(jobTemplateSpec)
		if err != nil {
			logger.Error(err, "Failed to reconcile job template", "name", jobTemplateSpec.Name)
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

	// Requeue every 60 seconds to ensure internal AWX state matches desired state
	return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
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

	// Create AWX client
	baseURL := fmt.Sprintf("https://%s", instance.Spec.Hostname)
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

// SetupWithManager sets up the controller with the Manager.
func (r *AWXInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&awxv1alpha1.AWXInstance{}).
		Complete(r)
}
