package awx

import (
	"fmt"

	awxv1alpha1 "github.com/derzufall/awx-k8s-operator/api/v1alpha1"
)

// JobTemplateManager handles AWX Job Template resources
type JobTemplateManager struct {
	client *Client
}

// NewJobTemplateManager creates a new JobTemplateManager
func NewJobTemplateManager(client *Client) *JobTemplateManager {
	return &JobTemplateManager{
		client: client,
	}
}

// GetJobTemplate retrieves a job template by name
func (jtm *JobTemplateManager) GetJobTemplate(name string) (map[string]interface{}, error) {
	log.Info("Fetching job template by name", "name", name)
	return jtm.client.FindObjectByName("job_templates", name)
}

// IsJobTemplateInDesiredState checks if the job template matches the desired specification
func (jtm *JobTemplateManager) IsJobTemplateInDesiredState(jobTemplate map[string]interface{}, jobTemplateSpec awxv1alpha1.JobTemplateSpec) bool {
	// Check name
	if name, ok := jobTemplate["name"].(string); !ok || name != jobTemplateSpec.Name {
		return false
	}

	// Check description
	if description, ok := jobTemplate["description"].(string); !ok || description != jobTemplateSpec.Description {
		return false
	}

	// Check playbook
	if playbook, ok := jobTemplate["playbook"].(string); !ok || playbook != jobTemplateSpec.Playbook {
		return false
	}

	// Check project
	project, ok := jobTemplate["project"]
	if !ok {
		return false
	}

	// Project can be an object or just an ID field, handle both cases
	projectObj, ok := project.(map[string]interface{})
	if ok {
		// Project is an object with a name field
		projectName, ok := projectObj["name"].(string)
		if !ok || projectName != jobTemplateSpec.ProjectName {
			return false
		}
	} else {
		// Project is an ID, we need to fetch the project to check its name
		projectID, ok := project.(float64)
		if !ok {
			return false
		}

		projectObj, err := jtm.client.GetObject("projects", int(projectID))
		if err != nil {
			return false
		}

		projectName, ok := projectObj["name"].(string)
		if !ok || projectName != jobTemplateSpec.ProjectName {
			return false
		}
	}

	// Check inventory
	inventory, ok := jobTemplate["inventory"]
	if !ok {
		return false
	}

	// Inventory can be an object or just an ID field, handle both cases
	inventoryObj, ok := inventory.(map[string]interface{})
	if ok {
		// Inventory is an object with a name field
		inventoryName, ok := inventoryObj["name"].(string)
		if !ok || inventoryName != jobTemplateSpec.InventoryName {
			return false
		}
	} else {
		// Inventory is an ID, we need to fetch the inventory to check its name
		inventoryID, ok := inventory.(float64)
		if !ok {
			return false
		}

		inventoryObj, err := jtm.client.GetObject("inventories", int(inventoryID))
		if err != nil {
			return false
		}

		inventoryName, ok := inventoryObj["name"].(string)
		if !ok || inventoryName != jobTemplateSpec.InventoryName {
			return false
		}
	}

	// Check extra vars if provided
	if jobTemplateSpec.ExtraVars != "" {
		if extraVars, ok := jobTemplate["extra_vars"].(string); !ok || extraVars != jobTemplateSpec.ExtraVars {
			return false
		}
	}

	return true
}

// EnsureJobTemplate ensures that a job template exists with the specified configuration
func (jtm *JobTemplateManager) EnsureJobTemplate(jobTemplateSpec awxv1alpha1.JobTemplateSpec) (map[string]interface{}, error) {
	log.Info("Ensuring job template exists with desired configuration", "name", jobTemplateSpec.Name)

	// First, check if job template exists
	jobTemplate, err := jtm.client.FindObjectByName("job_templates", jobTemplateSpec.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check if job template exists: %w", err)
	}

	// Find the project by name - required for job templates per AWX API docs
	log.Info("Finding associated project", "name", jobTemplateSpec.ProjectName)
	project, err := jtm.client.FindObjectByName("projects", jobTemplateSpec.ProjectName)
	if err != nil {
		return nil, fmt.Errorf("failed to find project %s: %w", jobTemplateSpec.ProjectName, err)
	}
	if project == nil {
		return nil, fmt.Errorf("project %s not found", jobTemplateSpec.ProjectName)
	}
	projectID, err := getObjectID(project)
	if err != nil {
		return nil, fmt.Errorf("failed to get project ID: %w", err)
	}

	// Find the inventory by name - required for job templates per AWX API docs
	log.Info("Finding associated inventory", "name", jobTemplateSpec.InventoryName)
	inventory, err := jtm.client.FindObjectByName("inventories", jobTemplateSpec.InventoryName)
	if err != nil {
		return nil, fmt.Errorf("failed to find inventory %s: %w", jobTemplateSpec.InventoryName, err)
	}
	if inventory == nil {
		return nil, fmt.Errorf("inventory %s not found", jobTemplateSpec.InventoryName)
	}
	inventoryID, err := getObjectID(inventory)
	if err != nil {
		return nil, fmt.Errorf("failed to get inventory ID: %w", err)
	}

	// Map job template spec to AWX API fields according to AWX API docs
	jobTemplateData := map[string]interface{}{
		"name":                     jobTemplateSpec.Name,
		"description":              jobTemplateSpec.Description,
		"project":                  projectID,
		"inventory":                inventoryID,
		"playbook":                 jobTemplateSpec.Playbook,
		"job_type":                 "run", // Default to 'run' if not specified
		"verbosity":                0,     // Default verbosity
		"ask_limit_on_launch":      false,
		"ask_inventory_on_launch":  false,
		"ask_credential_on_launch": false,
	}

	// Set extra vars if provided
	if jobTemplateSpec.ExtraVars != "" {
		jobTemplateData["extra_vars"] = jobTemplateSpec.ExtraVars
	}

	// Create or update job template
	if jobTemplate == nil {
		// Job template doesn't exist, create it
		log.Info("Creating AWX job template", "name", jobTemplateSpec.Name)
		jobTemplate, err = jtm.client.CreateObject("job_templates", jobTemplateData, "job_template")
		if err != nil {
			return nil, fmt.Errorf("failed to create job template: %w", err)
		}

		// Verify new job template has an ID
		if _, ok := jobTemplate["id"]; !ok {
			log.Error(nil, "Newly created job template missing ID field",
				"name", jobTemplateSpec.Name,
				"keys", getMapKeys(jobTemplate))
			return nil, fmt.Errorf("created job template '%s' has no ID field", jobTemplateSpec.Name)
		}

		log.Info("Successfully created job template",
			"name", jobTemplateSpec.Name,
			"id", jobTemplate["id"],
			"project", jobTemplateSpec.ProjectName,
			"inventory", jobTemplateSpec.InventoryName)
	} else {
		// Job template exists, update it
		id, err := getObjectID(jobTemplate)
		if err != nil {
			log.Error(err, "Cannot get ID from existing job template",
				"name", jobTemplateSpec.Name,
				"keys", getMapKeys(jobTemplate))
			return nil, fmt.Errorf("failed to get ID from existing job template '%s': %w", jobTemplateSpec.Name, err)
		}

		log.Info("Updating AWX job template",
			"name", jobTemplateSpec.Name,
			"id", id)
		jobTemplate, err = jtm.client.UpdateObject("job_templates", id, jobTemplateData)
		if err != nil {
			return nil, fmt.Errorf("failed to update job template: %w", err)
		}

		log.Info("Successfully updated job template",
			"name", jobTemplateSpec.Name,
			"id", id,
			"project", jobTemplateSpec.ProjectName,
			"inventory", jobTemplateSpec.InventoryName)
	}

	return jobTemplate, nil
}

// DeleteJobTemplate deletes a job template by name
func (jtm *JobTemplateManager) DeleteJobTemplate(name string) error {
	log.Info("Deleting job template", "name", name)

	jobTemplate, err := jtm.client.FindObjectByName("job_templates", name)
	if err != nil {
		return fmt.Errorf("failed to check if job template exists: %w", err)
	}

	if jobTemplate == nil {
		// Job template doesn't exist, nothing to do
		log.Info("Job template already deleted", "name", name)
		return nil
	}

	id, err := getObjectID(jobTemplate)
	if err != nil {
		return fmt.Errorf("failed to get job template ID: %w", err)
	}

	log.Info("Deleting AWX job template", "name", name, "id", id)
	err = jtm.client.DeleteObject("job_templates", id)
	if err != nil {
		return fmt.Errorf("failed to delete job template %s: %w", name, err)
	}

	log.Info("Successfully deleted job template", "name", name)
	return nil
}
