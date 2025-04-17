package awx

import (
	"fmt"

	awxv1alpha1 "github.com/derzufall/awx-k8s-operator/api/v1alpha1"
)

// InventoryManager handles AWX Inventory resources
type InventoryManager struct {
	client *Client
}

// NewInventoryManager creates a new InventoryManager
func NewInventoryManager(client *Client) *InventoryManager {
	return &InventoryManager{
		client: client,
	}
}

// GetInventory retrieves an inventory by name
func (im *InventoryManager) GetInventory(name string) (map[string]interface{}, error) {
	log.Info("Fetching inventory by name", "name", name)
	return im.client.FindObjectByName("inventories", name)
}

// IsInventoryInDesiredState checks if the inventory matches the desired specification
func (im *InventoryManager) IsInventoryInDesiredState(inventory map[string]interface{}, inventorySpec awxv1alpha1.InventorySpec) bool {
	// Check name
	if name, ok := inventory["name"].(string); !ok || name != inventorySpec.Name {
		return false
	}

	// Check description
	if description, ok := inventory["description"].(string); !ok || description != inventorySpec.Description {
		return false
	}

	// Check variables
	if inventorySpec.Variables != "" {
		if variables, ok := inventory["variables"].(string); !ok || variables != inventorySpec.Variables {
			return false
		}
	}

	// Check hosts
	if len(inventorySpec.Hosts) > 0 {
		// Get inventory ID for host operations
		inventoryID, err := getObjectID(inventory)
		if err != nil {
			return false
		}

		// Get existing hosts
		hostsEndpoint := fmt.Sprintf("inventories/%d/hosts", inventoryID)
		existingHosts, err := im.client.ListObjects(hostsEndpoint, nil)
		if err != nil {
			return false
		}

		// Build map of existing hosts for quick lookup
		existingHostMap := make(map[string]map[string]interface{})
		for _, host := range existingHosts {
			name, ok := host["name"].(string)
			if ok {
				existingHostMap[name] = host
			}
		}

		// Check if all desired hosts exist with correct configuration
		for _, hostSpec := range inventorySpec.Hosts {
			existingHost, exists := existingHostMap[hostSpec.Name]
			if !exists {
				// Host doesn't exist
				return false
			}

			// Check host configuration
			if !im.isHostInDesiredState(existingHost, hostSpec) {
				return false
			}
		}

		// Check if there are extra hosts that are not in the desired state
		if len(existingHosts) != len(inventorySpec.Hosts) {
			return false
		}
	}

	return true
}

// isHostInDesiredState checks if a host matches the desired specification
func (im *InventoryManager) isHostInDesiredState(host map[string]interface{}, hostSpec awxv1alpha1.HostSpec) bool {
	// Check name
	if name, ok := host["name"].(string); !ok || name != hostSpec.Name {
		return false
	}

	// Check description
	if description, ok := host["description"].(string); !ok || description != hostSpec.Description {
		return false
	}

	// Check variables
	if hostSpec.Variables != "" {
		if variables, ok := host["variables"].(string); !ok || variables != hostSpec.Variables {
			return false
		}
	}

	return true
}

// EnsureInventory ensures that an inventory exists with the specified configuration
func (im *InventoryManager) EnsureInventory(inventorySpec awxv1alpha1.InventorySpec) (map[string]interface{}, error) {
	log.Info("Ensuring inventory exists with desired configuration", "name", inventorySpec.Name)

	// First, check if inventory exists
	inventory, err := im.client.FindObjectByName("inventories", inventorySpec.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check if inventory exists: %w", err)
	}

	// Per AWX API docs, we need to set organization ID
	// Using default organization (ID 1) since it's not specified in our InventorySpec
	orgID := 1

	// Map inventory spec to AWX API fields
	inventoryData := map[string]interface{}{
		"name":         inventorySpec.Name,
		"description":  inventorySpec.Description,
		"variables":    inventorySpec.Variables,
		"organization": orgID,
	}

	var inventoryID int
	// Create or update inventory
	if inventory == nil {
		// Inventory doesn't exist, create it
		log.Info("Creating AWX inventory", "name", inventorySpec.Name, "organization", orgID)
		inventory, err = im.client.CreateObject("inventories", inventoryData)
		if err != nil {
			return nil, fmt.Errorf("failed to create inventory: %w", err)
		}

		// Verify new inventory has an ID
		if _, ok := inventory["id"]; !ok {
			log.Error(nil, "Newly created inventory missing ID field",
				"name", inventorySpec.Name,
				"keys", getMapKeys(inventory))
			return nil, fmt.Errorf("created inventory '%s' has no ID field", inventorySpec.Name)
		}

		log.Info("Successfully created inventory",
			"name", inventorySpec.Name,
			"id", inventory["id"])
	} else {
		// Inventory exists, update it
		inventoryID, err = getObjectID(inventory)
		if err != nil {
			log.Error(err, "Cannot get ID from existing inventory",
				"name", inventorySpec.Name,
				"keys", getMapKeys(inventory))
			return nil, fmt.Errorf("failed to get ID from existing inventory '%s': %w", inventorySpec.Name, err)
		}

		log.Info("Updating AWX inventory", "name", inventorySpec.Name, "id", inventoryID)
		inventory, err = im.client.UpdateObject("inventories", inventoryID, inventoryData)
		if err != nil {
			return nil, fmt.Errorf("failed to update inventory: %w", err)
		}

		log.Info("Successfully updated inventory",
			"name", inventorySpec.Name,
			"id", inventoryID)
	}

	// Get inventory ID for host operations
	inventoryID, err = getObjectID(inventory)
	if err != nil {
		return nil, fmt.Errorf("failed to get inventory ID for host operations in '%s': %w", inventorySpec.Name, err)
	}

	// Process hosts if defined
	if len(inventorySpec.Hosts) > 0 {
		log.Info("Reconciling inventory hosts",
			"inventory", inventorySpec.Name,
			"count", len(inventorySpec.Hosts))
		err = im.reconcileHosts(inventoryID, inventorySpec.Hosts)
		if err != nil {
			return nil, fmt.Errorf("failed to reconcile hosts for inventory '%s': %w", inventorySpec.Name, err)
		}
	}

	return inventory, nil
}

// reconcileHosts ensures that the hosts in the inventory match the desired state
func (im *InventoryManager) reconcileHosts(inventoryID int, desiredHosts []awxv1alpha1.HostSpec) error {
	// Per AWX API: use the related hosts endpoint for an inventory
	hostsEndpoint := fmt.Sprintf("inventories/%d/hosts", inventoryID)
	log.Info("Fetching existing hosts", "endpoint", hostsEndpoint)

	existingHosts, err := im.client.ListObjects(hostsEndpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to list existing hosts: %w", err)
	}

	// Build map of existing hosts for quick lookup
	existingHostMap := make(map[string]map[string]interface{})
	for _, host := range existingHosts {
		name, ok := host["name"].(string)
		if ok {
			existingHostMap[name] = host
		}
	}

	// Track desired host names to identify hosts to remove
	desiredHostNames := make(map[string]bool)

	// Create or update hosts according to AWX API docs
	for _, hostSpec := range desiredHosts {
		desiredHostNames[hostSpec.Name] = true

		// Map host spec to AWX API fields
		hostData := map[string]interface{}{
			"name":        hostSpec.Name,
			"description": hostSpec.Description,
			"inventory":   inventoryID,
			"variables":   hostSpec.Variables,
		}

		if existingHost, exists := existingHostMap[hostSpec.Name]; exists {
			// Update existing host
			hostID, err := getObjectID(existingHost)
			if err != nil {
				return fmt.Errorf("failed to get host ID: %w", err)
			}

			log.Info("Updating AWX host",
				"name", hostSpec.Name,
				"id", hostID,
				"inventory", inventoryID)
			_, err = im.client.UpdateObject("hosts", hostID, hostData)
			if err != nil {
				return fmt.Errorf("failed to update host %s: %w", hostSpec.Name, err)
			}
		} else {
			// Create new host
			log.Info("Creating AWX host",
				"name", hostSpec.Name,
				"inventory", inventoryID)
			_, err := im.client.CreateObject("hosts", hostData)
			if err != nil {
				return fmt.Errorf("failed to create host %s: %w", hostSpec.Name, err)
			}
		}
	}

	// Remove hosts that are not in the desired state
	// According to AWX API docs, we should use the DELETE method on each host
	for name, host := range existingHostMap {
		if !desiredHostNames[name] {
			hostID, err := getObjectID(host)
			if err != nil {
				return fmt.Errorf("failed to get host ID for deletion: %w", err)
			}

			log.Info("Deleting AWX host",
				"name", name,
				"id", hostID,
				"inventory", inventoryID)
			err = im.client.DeleteObject("hosts", hostID)
			if err != nil {
				return fmt.Errorf("failed to delete host %s: %w", name, err)
			}
		}
	}

	log.Info("Host reconciliation complete",
		"inventory", inventoryID,
		"hostCount", len(desiredHosts))
	return nil
}

// DeleteInventory deletes an inventory by name
func (im *InventoryManager) DeleteInventory(name string) error {
	inventory, err := im.client.FindObjectByName("inventories", name)
	if err != nil {
		return fmt.Errorf("failed to check if inventory exists: %w", err)
	}

	if inventory == nil {
		// Inventory doesn't exist, nothing to do
		return nil
	}

	id, err := getObjectID(inventory)
	if err != nil {
		return err
	}

	log.Info("Deleting AWX inventory", "name", name, "id", id)
	return im.client.DeleteObject("inventories", id)
}
