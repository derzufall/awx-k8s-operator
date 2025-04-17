package awx

import (
	"fmt"

	awxv1alpha1 "github.com/derzufall/awx-k8s-operator/api/v1alpha1"
)

// ProjectManager handles AWX Project resources
type ProjectManager struct {
	client *Client
}

// NewProjectManager creates a new ProjectManager
func NewProjectManager(client *Client) *ProjectManager {
	return &ProjectManager{
		client: client,
	}
}

// GetProject retrieves a project by name
func (pm *ProjectManager) GetProject(name string) (map[string]interface{}, error) {
	return pm.client.FindObjectByName("projects", name)
}

// IsProjectInDesiredState checks if the project matches the desired specification
func (pm *ProjectManager) IsProjectInDesiredState(project map[string]interface{}, projectSpec awxv1alpha1.ProjectSpec) bool {
	// Check name
	if name, ok := project["name"].(string); !ok || name != projectSpec.Name {
		return false
	}

	// Check description
	if description, ok := project["description"].(string); !ok || description != projectSpec.Description {
		return false
	}

	// Check SCM type
	if scmType, ok := project["scm_type"].(string); !ok || scmType != projectSpec.SCMType {
		return false
	}

	// Only check SCM URL if SCM type is not manual and URL is specified
	if projectSpec.SCMType != "manual" && projectSpec.SCMUrl != "" {
		if scmUrl, ok := project["scm_url"].(string); !ok || scmUrl != projectSpec.SCMUrl {
			return false
		}
	}

	// Check SCM branch if specified
	if projectSpec.SCMBranch != "" {
		if scmBranch, ok := project["scm_branch"].(string); !ok || scmBranch != projectSpec.SCMBranch {
			return false
		}
	}

	// Check SCM credential if specified
	if projectSpec.SCMCredential != "" {
		// Check if the credential relation exists
		credential, ok := project["credential"]
		if !ok {
			return false
		}

		// Get the credential object to check its name
		// This may require additional API calls, which could be optimized
		credentialObj, ok := credential.(map[string]interface{})
		if !ok {
			// In some cases the credential might be just an ID, not a full object
			// In that case, we'd need a separate API call to get the full object
			// This would require additional implementation
			return false
		}

		credName, ok := credentialObj["name"].(string)
		if !ok || credName != projectSpec.SCMCredential {
			return false
		}
	}

	return true
}

// EnsureProject ensures that a project exists with the specified configuration
func (pm *ProjectManager) EnsureProject(projectSpec awxv1alpha1.ProjectSpec) (map[string]interface{}, error) {
	// First, check if project exists
	project, err := pm.client.FindObjectByName("projects", projectSpec.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check if project exists: %w", err)
	}

	// Map project spec to AWX API fields
	projectData := map[string]interface{}{
		"name":        projectSpec.Name,
		"description": projectSpec.Description,
		"scm_type":    projectSpec.SCMType,
	}

	// Only set SCM URL if provided and SCM type is not manual
	if projectSpec.SCMType != "manual" && projectSpec.SCMUrl != "" {
		projectData["scm_url"] = projectSpec.SCMUrl
	}

	// Set SCM branch if provided
	if projectSpec.SCMBranch != "" {
		projectData["scm_branch"] = projectSpec.SCMBranch
	}

	// Set SCM credential if provided
	if projectSpec.SCMCredential != "" {
		credential, err := pm.client.FindObjectByName("credentials", projectSpec.SCMCredential)
		if err != nil {
			return nil, fmt.Errorf("failed to find SCM credential: %w", err)
		}

		if credential != nil {
			credentialID, ok := credential["id"]
			if ok {
				projectData["credential"] = credentialID
			}
		}
	}

	// Create or update project
	if project == nil {
		// Project doesn't exist, create it
		log.Info("Creating AWX project", "name", projectSpec.Name)
		project, err = pm.client.CreateObject("projects", projectData)
		if err != nil {
			return nil, fmt.Errorf("failed to create project: %w", err)
		}

		// Verify that we got a valid project back
		if project == nil {
			return nil, fmt.Errorf("received nil project after creation")
		}

		// Verify the project has the expected name
		if name, ok := project["name"].(string); !ok || name != projectSpec.Name {
			log.Error(nil, "Created project has unexpected name",
				"expected", projectSpec.Name,
				"actual", name,
				"keys", getMapKeys(project))
		}

		// Verify the project has an ID
		if _, ok := project["id"]; !ok {
			log.Error(nil, "Created project missing ID field",
				"name", projectSpec.Name,
				"keys", getMapKeys(project))
			return nil, fmt.Errorf("created project has no ID field")
		}

		// Log successful creation
		id, _ := getObjectID(project)
		log.Info("Successfully created AWX project", "name", projectSpec.Name, "id", id)

		return project, nil
	} else {
		// Project exists, update it
		id, err := getObjectID(project)
		if err != nil {
			log.Error(err, "Cannot get ID from existing project",
				"name", projectSpec.Name,
				"keys", getMapKeys(project))
			return nil, fmt.Errorf("failed to get ID from existing project '%s': %w", projectSpec.Name, err)
		}

		log.Info("Updating AWX project", "name", projectSpec.Name, "id", id)
		project, err = pm.client.UpdateObject("projects", id, projectData)
		if err != nil {
			return nil, fmt.Errorf("failed to update project: %w", err)
		}

		// Log successful update
		log.Info("Successfully updated AWX project", "name", projectSpec.Name, "id", id)

		return project, nil
	}
}

// DeleteProject deletes a project by name
func (pm *ProjectManager) DeleteProject(name string) error {
	project, err := pm.client.FindObjectByName("projects", name)
	if err != nil {
		return fmt.Errorf("failed to check if project exists: %w", err)
	}

	if project == nil {
		// Project doesn't exist, nothing to do
		return nil
	}

	id, err := getObjectID(project)
	if err != nil {
		return err
	}

	log.Info("Deleting AWX project", "name", name, "id", id)
	return pm.client.DeleteObject("projects", id)
}
