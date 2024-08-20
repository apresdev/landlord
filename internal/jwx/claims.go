package jwx

import (
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

// GetClaimString returns the string value of the claim with the given key.
func GetClaimString(tok *jwt.Token, key string) (string, error) {
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("could not convert claims to jwt.MapClaims")
	}
	v, ok := claims[key]
	if !ok {
		return "", fmt.Errorf("key %s not found", key)
	}
	vs, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("value for key %s is not a string", key)
	}
	return vs, nil
}

// GetClaimStrings returns the string values of the claim with the given key.
func GetClaimStrings(tok *jwt.Token, key string) ([]string, error) {
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("could not convert claims to jwt.MapClaims")
	}
	v, ok := claims[key]
	if !ok {
		return nil, fmt.Errorf("key %s not found", key)
	}
	switch v := v.(type) {
	case string:
		return []string{v}, nil
	case []string:
		return v, nil
	case []interface{}:
		var cs []string
		for _, a := range v {
			vs, ok := a.(string)
			if !ok {
				return nil, fmt.Errorf("value for key %s is not a string", key)
			}
			cs = append(cs, vs)
		}
		return cs, nil
	default:
		return nil, fmt.Errorf("value for key %s is not a string", key)
	}
}
