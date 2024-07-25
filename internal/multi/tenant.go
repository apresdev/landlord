package multi

import (
	"errors"
	"fmt"
	"regexp"
	"slices"

	"github.com/google/uuid"
)

const (
	RoleGuest             = "guest"
	RootTenantName        = "root"
	RootTenantDescription = "Root Tenant"
)

type TenantState string

const (
	TenantStateActive   TenantState = "active"
	TenantStateInactive TenantState = "inactive"
)

type Tenant struct {
	ID           uuid.UUID   `dynamodbav:"id"`
	State        TenantState `dynamodbav:"state"`
	Name         string      `dynamodbav:"name"`
	Description  string      `dynamodbav:"description"`
	Roles        []string    `dynamodbav:"roles"`
	UserPoolName string      `dynamodbav:"user_pool_name"`
	UserPoolID   string      `dynamodbav:"user_pool_id"`
}

func MustNewTenantID() uuid.UUID {
	return uuid.New()
}

func NewTenant() Tenant {
	return Tenant{
		State: TenantStateActive,
		Roles: []string{RoleGuest},
		ID:    MustNewTenantID(),
	}
}

func (ten *Tenant) AddRole(role string) error {
	if ten.HasRole(role) {
		return errors.New("role already exists")
	}
	ten.Roles = append(ten.Roles, role)
	return nil
}

func (ten *Tenant) HasRole(role string) bool {
	return slices.Contains(ten.Roles, role)
}

// GroupName returns the AWS Cognito group name that is used
// for the given role.
func (ten *Tenant) GroupName(role string) string {
	return fmt.Sprintf("tenant-%s-%s", ten.ID.String(), role)
}

const (
	minTenantNameLength = 3
	maxTenantNameLength = 32
)

func ValidateTenantName(name string) error {
	if len(name) < minTenantNameLength {
		return errors.New("tenant name must be at least 3 characters")
	}
	if len(name) > maxTenantNameLength {
		return errors.New("tenant name must be 32 characters or less")
	}
	if name[0] < 'a' || name[0] > 'z' {
		return errors.New("tenant name must start with a letter")
	}
	matched, err := regexp.MatchString(`^[a-z0-9-]+$`, name)
	if err != nil {
		return err
	}
	if !matched {
		return errors.New("tenant name must be all lowercase and contain only letters, numbers, and hyphens")
	}
	return nil
}
