package awx

import (
	"fmt"

	awxv1alpha1 "github.com/yourusername/awx-operator/api/v1alpha1"
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
	// First, check if inventory exists
	inventory, err := im.client.FindObjectByName("inventories", inventorySpec.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check if inventory exists: %w", err)
	}

	// Map inventory spec to AWX API fields
	inventoryData := map[string]interface{}{
		"name":        inventorySpec.Name,
		"description": inventorySpec.Description,
		"variables":   inventorySpec.Variables,
	}

	var inventoryID int
	// Create or update inventory
	if inventory == nil {
		// Inventory doesn't exist, create it
		log.Info("Creating AWX inventory", "name", inventorySpec.Name)
		inventory, err = im.client.CreateObject("inventories", inventoryData)
		if err != nil {
			return nil, fmt.Errorf("failed to create inventory: %w", err)
		}
	} else {
		// Inventory exists, update it
		inventoryID, err = getObjectID(inventory)
		if err != nil {
			return nil, err
		}

		log.Info("Updating AWX inventory", "name", inventorySpec.Name, "id", inventoryID)
		inventory, err = im.client.UpdateObject("inventories", inventoryID, inventoryData)
		if err != nil {
			return nil, fmt.Errorf("failed to update inventory: %w", err)
		}
	}

	// Get inventory ID for host operations
	inventoryID, err = getObjectID(inventory)
	if err != nil {
		return nil, err
	}

	// Process hosts if defined
	if len(inventorySpec.Hosts) > 0 {
		err = im.reconcileHosts(inventoryID, inventorySpec.Hosts)
		if err != nil {
			return nil, fmt.Errorf("failed to reconcile hosts: %w", err)
		}
	}

	return inventory, nil
}

// reconcileHosts ensures that the hosts in the inventory match the desired state
func (im *InventoryManager) reconcileHosts(inventoryID int, desiredHosts []awxv1alpha1.HostSpec) error {
	// Get existing hosts
	hostsEndpoint := fmt.Sprintf("inventories/%d/hosts", inventoryID)
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

	// Create or update hosts
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
				return err
			}

			log.Info("Updating AWX host", "name", hostSpec.Name, "id", hostID)
			_, err = im.client.UpdateObject("hosts", hostID, hostData)
			if err != nil {
				return fmt.Errorf("failed to update host %s: %w", hostSpec.Name, err)
			}
		} else {
			// Create new host
			log.Info("Creating AWX host", "name", hostSpec.Name, "inventory", inventoryID)
			_, err := im.client.CreateObject("hosts", hostData)
			if err != nil {
				return fmt.Errorf("failed to create host %s: %w", hostSpec.Name, err)
			}
		}
	}

	// Remove hosts that are not in the desired state
	for name, host := range existingHostMap {
		if !desiredHostNames[name] {
			hostID, err := getObjectID(host)
			if err != nil {
				return err
			}

			log.Info("Deleting AWX host", "name", name, "id", hostID)
			err = im.client.DeleteObject("hosts", hostID)
			if err != nil {
				return fmt.Errorf("failed to delete host %s: %w", name, err)
			}
		}
	}

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
