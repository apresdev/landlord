package perms

import (
	"context"
	"embed"
	"fmt"

	"github.com/open-policy-agent/opa/rego"

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

func (o Operation) String() string {
	switch o {
	case OperationRead:
		return "read"
	case OperationWrite:
		return "write"
	case OperationDelete:
		return "delete"
	default:
		return "unknown"
	}
}

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

const ResourceNameAny = "__any__"

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

//go:embed policies/*
var policiesFS embed.FS

var builtinPolicy rego.PreparedEvalQuery

func getBuiltinModule() string {
	b, err := policiesFS.ReadFile("policies/builtin.rego")
	if err != nil {
		zap.L().Fatal("failed to read policy module", zap.Error(err))
	}
	return string(b)
}

func init() {
	ctx := context.Background()
	query, err := rego.New(
		rego.Query("x = data.landlord.policy.allow"),
		rego.Module("landlord.policy", getBuiltinModule()),
	).PrepareForEval(ctx)
	if err != nil {
		zap.L().Fatal("failed to prepare policy query", zap.Error(err))
	}
	builtinPolicy = query
}

// Eval evaluates our permission policy. It returns an Effect that indicates
// whether the operation is allowed or denied.
func Eval(ctx context.Context, app App, p *Principal, op Operation, r *Resource) (*Effect, error) {
	principalTenant, err := p.ResolveTenant(ctx, app)
	if err != nil {
		return nil, err
	}
	principalUser, err := p.ResolveUser(ctx, app)
	if err != nil {
		return nil, err
	}

	resource := map[string]any{
		"kind": r.Kind.String(),
		"name": r.Name,
	}
	if r.TenantOwner != nil {
		resource["tenantOwner"] = map[string]string{
			"id":   r.TenantOwner.ID.String(),
			"name": r.TenantOwner.Name,
		}
	}
	if r.UserOwner != nil {
		resource["userOwner"] = map[string]string{
			"id":       r.UserOwner.ID.String(),
			"username": r.UserOwner.Username,
			"role":     r.UserOwner.Role,
			"status":   string(r.UserOwner.Status),
		}
	}
	input := map[string]any{
		"principal": map[string]map[string]string{
			"user": {
				"id":        principalUser.ID.String(),
				"username":  principalUser.Username,
				"role":      principalUser.Role,
				"status":    string(principalUser.Status),
				"mfaStatus": string(principalUser.MFAStatus),
				"mfaPolicy": principalUser.MFAPolicy.String(),
			},
			"tenant": {
				"id":   principalTenant.ID.String(),
				"name": principalTenant.Name,
			},
		},
		"operation": op.String(),
		"resource":  resource,
	}

	results, err := builtinPolicy.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no results")
	}
	result, ok := results[0].Bindings["x"].(bool)
	if !ok {
		return nil, fmt.Errorf("unexpected result type")
	}
	return &Effect{allowed: result}, nil
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
