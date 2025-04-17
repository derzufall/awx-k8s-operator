package awx

import (
	"fmt"
	"strconv"
)

// getObjectID extracts the ID from an AWX API object
func getObjectID(obj map[string]interface{}) (int, error) {
	idVal, ok := obj["id"]
	if !ok {
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
