package perms

import (
	"context"
	"fmt"

	"github.com/apresdev/landlord/internal/db"
	"github.com/apresdev/landlord/internal/multi"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Operation int

const (
	OperationUnknown Operation = iota
	OperationRead
	OperationWrite
	OperationDelete
)

type ResourceKind int

const (
	ResourceKindUnknown ResourceKind = iota
	ResourceKindImpersonation
	ResourceKindInvitation
	ResourceKindTenant
	ResourceKindTenantDescription
	ResourceKindTenantRoles
	ResourceKindTenantState
	ResourceKindUserMFAPolicy
	ResourceKindUserRole
	ResourceKindUserStatus
)

func (k ResourceKind) String() string {
	switch k {
	case ResourceKindImpersonation:
		return "impersonation"
	case ResourceKindInvitation:
		return "invitation"
	case ResourceKindTenant:
		return "tenant"
	case ResourceKindTenantDescription:
		return "tenant-description"
	case ResourceKindTenantRoles:
		return "tenant-roles"
	case ResourceKindTenantState:
		return "tenant-state"
	case ResourceKindUserMFAPolicy:
		return "user-mfa-policy"
	case ResourceKindUserRole:
		return "user-role"
	case ResourceKindUserStatus:
		return "user-status"
	default:
		return "unknown"
	}
}

type Resource struct {
	Kind        ResourceKind
	Name        string
	TenantOwner *multi.Tenant
	UserOwner   *multi.User
}

const ResourceNameAny = "_any"

func NewResource(kind ResourceKind, resourceName string, ten *multi.Tenant, user *multi.User) *Resource {
	return &Resource{
		Kind:        kind,
		Name:        resourceName,
		TenantOwner: ten,
		UserOwner:   user,
	}
}

func (r *Resource) String() string {
	return fmt.Sprintf("%s@%s@:%s@%s",
		r.UserOwner, r.TenantOwner, r.Name, r.Kind.String())
}

type Effect struct {
	allowed bool
}

func (e *Effect) IsAllowed() bool {
	return e.allowed
}

func (e *Effect) IsDenied() bool {
	return !e.allowed
}

// Eval evaluates our permission policy. It returns an Effect that indicates
// whether the operation is allowed or denied.
func Eval(ctx context.Context, app App, p *Principal, op Operation, r *Resource) (*Effect, error) {
	// TODO: for now, we allow all operations as long as either the Principal is
	// a member of the root tenant or the Principal is a member of the tenant
	// that owns the resource in question.
	for _, role := range p.Roles.Roles {
		if role.Tenant == app.RootTenantID() {
			return &Effect{allowed: true}, nil
		}
		if role.Tenant == r.TenantOwner.ID {
			return &Effect{allowed: true}, nil
		}
	}
	return &Effect{allowed: false}, nil
}

type App interface {
	DDB() db.AWSDynamoDB
	RootTenantID() uuid.UUID
}

// IsRoleAuthorizedForTenant checks if the given role is authorized for the given
// tenant.
func IsRoleAuthorizedForTenant(
	rootTenantID uuid.UUID,
	ten *multi.Tenant,
	role *multi.Role) bool {
	if ten == nil || role == nil {
		zap.L().Error("expected non-nil tenant and role",
			zap.Any("tenant", ten), zap.Any("role", role))
		return false
	}
	return role.Tenant == ten.ID || role.Tenant == rootTenantID
}

// IsAuthorizedToReadTenant returns true if the given set of roles
// are authorized for the given tenant.
func IsAuthorizedToReadTenant(
	rootTenantID uuid.UUID,
	ten *multi.Tenant,
	roles *multi.Roles) bool {

	if ten == nil || roles == nil {
		zap.L().Error("expected non-nil tenant and roles",
			zap.Any("tenant", ten), zap.Any("roles", roles))
		return false
	}

	for _, role := range roles.Roles {
		if IsRoleAuthorizedForTenant(rootTenantID, ten, role) {
			return true
		}
	}
	return false
}
