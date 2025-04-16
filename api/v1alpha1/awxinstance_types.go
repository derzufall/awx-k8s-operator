package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AWXInstanceSpec defines the desired state of AWXInstance
type AWXInstanceSpec struct {
	// AdminUser is the AWX admin username
	// +kubebuilder:validation:Required
	AdminUser string `json:"adminUser"`

	// AdminPassword is the AWX admin password
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=5
	AdminPassword string `json:"adminPassword"`

	// AdminEmail is the AWX admin email
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Format=email
	AdminEmail string `json:"adminEmail"`

	// Hostname is the hostname to access AWX UI
	// +kubebuilder:validation:Required
	Hostname string `json:"hostname"`

	// Protocol is the protocol to use for the AWX connection (http or https)
	// +kubebuilder:validation:Enum=http;https
	// +kubebuilder:default=https
	// +optional
	Protocol string `json:"protocol,omitempty"`

	// ExternalInstance indicates this is an existing AWX instance that should be managed but not created
	// +optional
	ExternalInstance bool `json:"externalInstance,omitempty"`

	// Replicas is the number of AWX workers to deploy
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	Replicas int32 `json:"replicas,omitempty"`

	// Projects defines the AWX projects to create
	// +optional
	Projects []ProjectSpec `json:"projects,omitempty"`

	// Inventories defines the AWX inventories to create
	// +optional
	Inventories []InventorySpec `json:"inventories,omitempty"`

	// JobTemplates defines the AWX job templates to create
	// +optional
	JobTemplates []JobTemplateSpec `json:"jobTemplates,omitempty"`
}

// ProjectSpec defines an AWX Project
type ProjectSpec struct {
	// Name is the project name
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Description of the project
	// +optional
	Description string `json:"description,omitempty"`

	// SCMType is the source control type (git, svn, etc)
	// +kubebuilder:validation:Enum=git;svn;manual
	// +kubebuilder:default=git
	SCMType string `json:"scmType,omitempty"`

	// SCMUrl is the source control URL
	// +optional
	SCMUrl string `json:"scmUrl,omitempty"`

	// SCMBranch is the source control branch
	// +optional
	// +kubebuilder:default=main
	SCMBranch string `json:"scmBranch,omitempty"`

	// SCMCredential is the name of the credential to use for SCM
	// +optional
	SCMCredential string `json:"scmCredential,omitempty"`
}

// InventorySpec defines an AWX Inventory
type InventorySpec struct {
	// Name is the inventory name
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Description of the inventory
	// +optional
	Description string `json:"description,omitempty"`

	// Variables is the inventory variables in YAML format
	// +optional
	Variables string `json:"variables,omitempty"`

	// Hosts defines the hosts in this inventory
	// +optional
	Hosts []HostSpec `json:"hosts,omitempty"`
}

// HostSpec defines a host in an inventory
type HostSpec struct {
	// Name is the host name
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Description of the host
	// +optional
	Description string `json:"description,omitempty"`

	// Variables is the host variables in YAML format
	// +optional
	Variables string `json:"variables,omitempty"`
}

// JobTemplateSpec defines an AWX Job Template
type JobTemplateSpec struct {
	// Name is the job template name
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Description of the job template
	// +optional
	Description string `json:"description,omitempty"`

	// ProjectName is the name of the project this job template belongs to
	// +kubebuilder:validation:Required
	ProjectName string `json:"projectName"`

	// InventoryName is the name of the inventory this job template uses
	// +kubebuilder:validation:Required
	InventoryName string `json:"inventoryName"`

	// Playbook is the name of the playbook to run
	// +kubebuilder:validation:Required
	Playbook string `json:"playbook"`

	// ExtraVars is the extra variables for the job template in YAML format
	// +optional
	ExtraVars string `json:"extraVars,omitempty"`
}

// AWXInstanceStatus defines the observed state of AWXInstance
type AWXInstanceStatus struct {
	// Conditions represent the latest available observations of the AWXInstance's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ProjectStatuses contains the reconciliation status of each project
	// +optional
	ProjectStatuses map[string]string `json:"projectStatuses,omitempty"`

	// InventoryStatuses contains the reconciliation status of each inventory
	// +optional
	InventoryStatuses map[string]string `json:"inventoryStatuses,omitempty"`

	// JobTemplateStatuses contains the reconciliation status of each job template
	// +optional
	JobTemplateStatuses map[string]string `json:"jobTemplateStatuses,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Hostname",type="string",JSONPath=".spec.hostname"
//+kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// AWXInstance is the Schema for the awxinstances API
type AWXInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AWXInstanceSpec   `json:"spec,omitempty"`
	Status AWXInstanceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AWXInstanceList contains a list of AWXInstance
type AWXInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AWXInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AWXInstance{}, &AWXInstanceList{})
}
