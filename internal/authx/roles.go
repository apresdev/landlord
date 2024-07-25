package authx

import (
	"fmt"
	"strings"

	"github.com/apresdev/landlord/internal/multi"

	"github.com/google/uuid"
)

type Role struct {
	Name   string
	Tenant uuid.UUID
}

type Roles struct {
	Roles []Role
}

func ParseRolesFromGroups(groups []string) (Roles, error) {
	result := Roles{Roles: make([]Role, 0)}
	for _, group := range groups {
		role, err := ParseRoleFromGroup(group)
		if err != nil {
			return Roles{}, err
		}
		result.Roles = append(result.Roles, role)
	}
	return result, nil
}

func ParseRoleFromGroup(group string) (Role, error) {
	parts := strings.Split(group, "-")
	if len(parts) != 7 {
		return Role{}, fmt.Errorf("invalid group: %s", group)
	}
	tenantID, err := uuid.Parse(strings.Join(parts[1:6], "-"))
	if err != nil {
		return Role{}, fmt.Errorf("invalid tenant ID: %s", group)
	}
	return Role{Name: parts[6], Tenant: tenantID}, nil
}

func IsAuthorizedForTenant(ten *multi.Tenant, roles Roles) bool {
	for _, role := range roles.Roles {
		if role.Tenant == ten.ID {
			return true
		}
	}
	return false
}
