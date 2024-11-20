package multi

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

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
