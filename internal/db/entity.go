package db

import (
	"fmt"
	"strings"
)

const (
	EntityNameGlobal = "Global"
	EntityNameTenant = "Tenant"
	EntityNameUser   = "User"
	EntitySep        = ":"
)

func entityID(name, id string) string {
	return fmt.Sprintf("%s%s%s", name, EntitySep, id)
}

func GlobalEntityID(id string) string {
	return entityID(EntityNameGlobal, id)
}

func TenantEntityID(id string) string {
	return entityID(EntityNameTenant, id)
}

func UserEntityID(id string) string {
	return entityID(EntityNameUser, id)
}

func StripEntityID(prefix, s string) (string, error) {
	prefix = prefix + EntitySep
	s, ok := strings.CutPrefix(s, prefix)
	if !ok {
		return "", fmt.Errorf(
			"invalid entity ID prefix: %s expecting prefix: %s", s, prefix)
	}
	return s, nil
}
