package jwx

import "fmt"

func strFromMap(m map[string]interface{}, key string) (string, error) {
	v, ok := m[key]
	if !ok {
		return "", fmt.Errorf("key %s not found", key)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("value for key %s is not a string", key)
	}
	return s, nil
}
