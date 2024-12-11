package multi

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/google/uuid"
)

const (
	RoleAdmin             = "admin"
	RoleGuest             = "guest"
	RootTenantName        = "root"
	RootTenantDescription = "Root Tenant"
	// MaxRoleNameLength is required as there is a limit to the name of a Cognito
	// group. And we format our Cognito groups as "tenant-<tenantID>-<role>". We
	// limit both role names and tenant names to 32 characters.
	MaxRoleNameLength = 32
)

var BuiltInRoles = []string{RoleAdmin, RoleGuest}

// IsBuiltinRole returns true if role is one of the built-in roles
func IsBuiltinRole(role string) bool {
	return slices.Contains(BuiltInRoles, role)
}

// ValidateRoleName validates a role name, returning a descriptive error if the
// name is invalid.
func ValidateRoleName(s string) error {
	if s == "" {
		return apperr.NewBadRequest("role name cannot be empty")
	}
	if len(s) > MaxRoleNameLength {
		return apperr.NewBadRequest(
			fmt.Sprintf("role name must be %d characters or less", MaxRoleNameLength))
	}
	// role names must be lowercase and contain only letters, numbers, underscores, and hyphens
	matched, err := regexp.MatchString(`^[a-z0-9_-]+$`, s)
	if err != nil {
		return apperr.NewBadRequestE(err)
	}
	if !matched {
		return apperr.NewBadRequest(
			"role name must be all lowercase and contain only letters, numbers, underscores, and hyphens")
	}
	return nil
}

// Role represents a role within a tenant. At the time of this writing,
// each user is restricted to a single role within a tenant. Certain users
// have permission to impersonate a user or role from other tenants.
type Role struct {
	Name   string
	Tenant uuid.UUID
}

func NewRole(name string, tenantID uuid.UUID) *Role {
	return &Role{Name: name, Tenant: tenantID}
}

func (r Role) String() string {
	return fmt.Sprintf("tenant-%s-%s", r.Tenant, r.Name)
}

type Roles struct {
	Roles []*Role
}

func NewRoles() *Roles {
	return &Roles{Roles: make([]*Role, 0)}
}

func (r Roles) String() string {
	result := make([]string, len(r.Roles))
	for i, role := range r.Roles {
		result[i] = role.String()
	}
	return strings.Join(result, ", ")
}
