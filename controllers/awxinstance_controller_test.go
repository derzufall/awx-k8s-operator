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
	"testing"

	awxv1alpha1 "github.com/derzufall/awx-k8s-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestReconcileInternalChangesWithNilMaps verifies that the reconcileInternalChanges
// method properly initializes nil status maps to prevent the nil map panic.
func TestReconcileInternalChangesWithNilMaps(t *testing.T) {
	// Create a fake client
	scheme := runtime.NewScheme()
	_ = awxv1alpha1.AddToScheme(scheme)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	// Create a reconciler with the fake client
	reconciler := &AWXInstanceReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

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
			// Add a project to trigger the map access
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

	// Create a mock AWX client that will return nothing for the GetProject call
	mockClient := &MockClient{
		GetProjectFunc: func(name string) (map[string]interface{}, error) {
			return nil, nil
		},
		EnsureProjectFunc: func(spec awxv1alpha1.ProjectSpec) (map[string]interface{}, error) {
			return map[string]interface{}{
				"id":   float64(1),
				"name": spec.Name,
			}, nil
		},
	}

	// Call reconcileInternalChanges with the instance that has nil status maps
	changed, err := reconciler.reconcileInternalChanges(context.Background(), instance, mockClient)

	// Verify there was no error
	assert.NoError(t, err)

	// Verify a change was detected (since the project didn't exist)
	assert.True(t, changed)

	// Verify the status maps were initialized
	assert.NotNil(t, instance.Status.ProjectStatuses)
	assert.NotNil(t, instance.Status.InventoryStatuses)
	assert.NotNil(t, instance.Status.JobTemplateStatuses)

	// Verify the project status was set
	assert.Equal(t, "Reconciled (corrected internal changes)", instance.Status.ProjectStatuses["test-project"])
}

// MockClient is a simple implementation of the AWX client for testing
type MockClient struct {
	GetProjectFunc        func(name string) (map[string]interface{}, error)
	EnsureProjectFunc     func(spec awxv1alpha1.ProjectSpec) (map[string]interface{}, error)
	GetInventoryFunc      func(name string) (map[string]interface{}, error)
	EnsureInventoryFunc   func(spec awxv1alpha1.InventorySpec) (map[string]interface{}, error)
	GetJobTemplateFunc    func(name string) (map[string]interface{}, error)
	EnsureJobTemplateFunc func(spec awxv1alpha1.JobTemplateSpec) (map[string]interface{}, error)
	TestConnectionFunc    func() error
}

func (m *MockClient) GetProject(name string) (map[string]interface{}, error) {
	if m.GetProjectFunc != nil {
		return m.GetProjectFunc(name)
	}
	return nil, nil
}

func (m *MockClient) EnsureProject(spec awxv1alpha1.ProjectSpec) (map[string]interface{}, error) {
	if m.EnsureProjectFunc != nil {
		return m.EnsureProjectFunc(spec)
	}
	return nil, nil
}

func (m *MockClient) GetInventory(name string) (map[string]interface{}, error) {
	if m.GetInventoryFunc != nil {
		return m.GetInventoryFunc(name)
	}
	return nil, nil
}

func (m *MockClient) EnsureInventory(spec awxv1alpha1.InventorySpec) (map[string]interface{}, error) {
	if m.EnsureInventoryFunc != nil {
		return m.EnsureInventoryFunc(spec)
	}
	return nil, nil
}

func (m *MockClient) GetJobTemplate(name string) (map[string]interface{}, error) {
	if m.GetJobTemplateFunc != nil {
		return m.GetJobTemplateFunc(name)
	}
	return nil, nil
}

func (m *MockClient) EnsureJobTemplate(spec awxv1alpha1.JobTemplateSpec) (map[string]interface{}, error) {
	if m.EnsureJobTemplateFunc != nil {
		return m.EnsureJobTemplateFunc(spec)
	}
	return nil, nil
}

func (m *MockClient) TestConnection() error {
	if m.TestConnectionFunc != nil {
		return m.TestConnectionFunc()
	}
	return nil
}
