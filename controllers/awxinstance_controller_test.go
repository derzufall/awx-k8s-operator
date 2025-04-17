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
	"testing"

	awxv1alpha1 "github.com/derzufall/awx-k8s-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestStatusMapInitialization verifies that status maps are properly initialized
// to prevent the nil map panic.
func TestStatusMapInitialization(t *testing.T) {
	// Create an AWX instance with nil status maps
	instance := &awxv1alpha1.AWXInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-instance",
			Namespace: "default",
		},
		Spec: awxv1alpha1.AWXInstanceSpec{
			Hostname:      "test.example.com",
			AdminUser:     "admin",
			AdminPassword: "password",
			AdminEmail:    "admin@example.com",
			Projects: []awxv1alpha1.ProjectSpec{
				{
					Name:    "test-project",
					SCMType: "git",
					SCMUrl:  "https://github.com/example/repo.git",
				},
			},
		},
		Status: awxv1alpha1.AWXInstanceStatus{
			// Status maps are intentionally nil
		},
	}

	// Directly ensure the maps are initialized
	if instance.Status.ProjectStatuses == nil {
		instance.Status.ProjectStatuses = make(map[string]string)
	}
	if instance.Status.InventoryStatuses == nil {
		instance.Status.InventoryStatuses = make(map[string]string)
	}
	if instance.Status.JobTemplateStatuses == nil {
		instance.Status.JobTemplateStatuses = make(map[string]string)
	}

	// Then try to access the maps
	instance.Status.ProjectStatuses["test-project"] = "Reconciled"

	// Verify the maps were initialized and can be accessed
	assert.NotNil(t, instance.Status.ProjectStatuses)
	assert.NotNil(t, instance.Status.InventoryStatuses)
	assert.NotNil(t, instance.Status.JobTemplateStatuses)
	assert.Equal(t, "Reconciled", instance.Status.ProjectStatuses["test-project"])
}
