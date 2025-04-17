package awx

import (
	"fmt"
	"strconv"
)

// getObjectID extracts the ID from an AWX API object
func getObjectID(obj map[string]interface{}) (int, error) {
	idVal, ok := obj["id"]
	if !ok {
		// Log additional details about the object to help debug
		log.Error(nil, "Object missing ID field", "object_keys", getMapKeys(obj))
		return 0, fmt.Errorf("object has no ID field")
	}

	// Handle different types of ID (float64 from JSON or int)
	switch id := idVal.(type) {
	case float64:
		return int(id), nil
	case int:
		return id, nil
	case string:
		return strconv.Atoi(id)
	default:
		return 0, fmt.Errorf("unexpected ID type: %T", idVal)
	}
}

// getMapKeys returns the keys of a map as a slice for logging
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
