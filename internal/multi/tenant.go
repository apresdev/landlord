package multi

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"time"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/google/uuid"
)

type TenantState string

const (
	TenantStateActive   TenantState = "active"
	TenantStateInactive TenantState = "inactive"
)

func ParseTenantState(state string) (TenantState, error) {
	switch state {
	case string(TenantStateActive):
		return TenantStateActive, nil
	case string(TenantStateInactive):
		return TenantStateInactive, nil
	default:
		return "", errors.New("invalid tenant state")
	}
}

func (ts TenantState) String() string {
	return string(ts)
}

type Tenant struct {
	ID             uuid.UUID   `dynamodbav:"-"` // sk -- pk is 'Global:*'
	Name           string      `dynamodbav:"-"` // altName
	State          TenantState `dynamodbav:"state"`
	Description    string      `dynamodbav:"description"`
	Roles          []string    `dynamodbav:"roles"`
	UserPoolID     string      `dynamodbav:"userPoolID"`
	UserPoolName   string      `dynamodbav:"userPoolName"`
	SubTenantRoles []string    `dynamodbav:"subTenantRoles"`
	EmailDomains   []string    `dynamodbav:"emailDomains"`
	CreatedAt      string      `dynamodbav:"createdAt"`
	RawProfile     []byte      `dynamodbav:"rawProfile"`
	profile        *Profile    `dynamodbav:"-"`
}

func MustNewTenantID() uuid.UUID {
	return uuid.New()
}

func NewTenant(name string) *Tenant {
	return &Tenant{
		Name:           name,
		State:          TenantStateActive,
		Roles:          BuiltInRoles,
		ID:             MustNewTenantID(),
		SubTenantRoles: BuiltInRoles,
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
	}
}

func (ten *Tenant) IsRoot() bool {
	return ten.Name == RootTenantName
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

func (ten *Tenant) HasSubTenantRole(role string) bool {
	return slices.Contains(ten.SubTenantRoles, role)
}

func (ten *Tenant) AddSubTenantRole(role string) error {
	if ten.HasSubTenantRole(role) {
		return errors.New("role already exists")
	}
	ten.SubTenantRoles = append(ten.SubTenantRoles, role)
	return nil
}

func (ten *Tenant) RemoveSubTenantRole(role string) error {
	idx := slices.Index(ten.SubTenantRoles, role)
	if idx == -1 {
		return errors.New("role not found")
	}
	if role == RoleGuest || role == RoleAdmin {
		return errors.New("cannot remove guest or admin role")
	}
	ten.SubTenantRoles = append(ten.SubTenantRoles[:idx], ten.SubTenantRoles[idx+1:]...)
	return nil
}

func (ten *Tenant) UsingDefaultSubTenantRoles() bool {
	return slices.Equal(ten.SubTenantRoles, BuiltInRoles)
}

func (ten *Tenant) GetUserProfileDefinition() *ProfileDefinition {
	// TODO: make this configurable eventually -- each tenant can, in theory,
	// have its own profile definition. We definitely want each deployment to
	// have its own profile definition. For now, we have a pre-baked profile
	// definition.
	return &BasicUserProfileFieldDefinition
}

func (ten *Tenant) GetTenantProfileDefinition() *ProfileDefinition {
	return &BasicTenantProfileFieldDefinition
}

func (ten *Tenant) Profile() *Profile {
	return lazyUnmarshalProfile(ten.profile, ten.RawProfile)
}

func (ten *Tenant) PrepareProfile() error {
	if ten.profile == nil {
		return nil
	}
	var err error
	ten.RawProfile, err = json.Marshal(ten.profile)
	if err != nil {
		return apperr.NewInternalE(
			fmt.Errorf("failed to marshal profile: %w", err))
	}
	return nil
}

func (ten *Tenant) ProfileFieldValue(name string) string {
	ten.profile = lazyUnmarshalProfile(ten.profile, ten.RawProfile)
	f := ten.profile.GetField(name)
	if f == nil {
		return EmptyProfileValue
	}
	return f.Value
}

func (ten *Tenant) ProfileField(name string) *ProfileField {
	ten.profile = lazyUnmarshalProfile(ten.profile, ten.RawProfile)
	return ten.profile.GetField(name)
}

func (ten *Tenant) UpdateProfileField(name string, format ProfileFieldFormat, value string) {
	ten.profile = lazyUnmarshalProfile(ten.profile, ten.RawProfile)
	ten.profile.SetField(name, format, value)
}

func (ten *Tenant) ProfileFields() []ProfileField {
	ten.profile = lazyUnmarshalProfile(ten.profile, ten.RawProfile)
	return ten.profile.Fields
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

const maxTenantDescriptionLength = 256

func ValidateTenantDescription(s string) error {
	if len(s) > maxTenantDescriptionLength {
		return fmt.Errorf(
			"description must be %d characters or less", maxTenantDescriptionLength)
	}
	return nil
}
