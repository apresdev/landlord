package authx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"slices"

	"github.com/aws/aws-sdk-go-v2/service/sts"
	"go.uber.org/zap"

	"github.com/google/uuid"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	cogidptypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/ettle/strcase"
)

const (
	RootTenantName     = "root"
	RootTenantFullName = "Root Tenant"
)

type TenantHandle struct {
	ID   uuid.UUID
	Name string
}

func ParseTenantHandle(id string, name string) (TenantHandle, error) {
	if id == "" && name == "" {
		return TenantHandle{},
			errors.New("tenant ID or tenant name must be set")
	}
	if id != "" && name != "" {
		return TenantHandle{},
			errors.New("tenant ID and tenant name cannot both be set")
	}
	if id != "" {
		u, err := uuid.Parse(id)
		if err != nil {
			return TenantHandle{}, err
		}
		return TenantHandle{
			ID: u,
		}, nil
	} else {
		return TenantHandle{
			Name: name,
		}, nil
	}
}

func (h TenantHandle) IsSet() bool {
	return h.ID.String() != "" || h.Name != ""
}

func MustNewTenantID() uuid.UUID {
	return uuid.New()
}

type UserStatus string

const (
	UserStatusInactive UserStatus = "inactive"
	UserStatusInvited  UserStatus = "invited"
	UserStatusActive   UserStatus = "active"
)

type User struct {
	Email  string     `json:"email"`
	Role   string     `json:"role"`
	Status UserStatus `json:"status"`
}

type Tenant struct {
	ID                    uuid.UUID `json:"id"`
	Name                  string    `json:"name"`
	FullName              string    `json:"full_name"`
	CustomAuthDomain      string    `json:"custom_auth_domain"`
	AllowedEmailDomains   []string  `json:"allowed_email_domains"`
	ExclusiveEmailDomains []string  `json:"exclusive_email_domains"`
	Users                 []User    `json:"users"`
	UserPoolName          string    `json:"user_pool_name"`
	UserPoolID            string    `json:"user_pool_id"`
}

func NewTenant() Tenant {
	return Tenant{
		ID: MustNewTenantID(),
	}
}

func (ten *Tenant) User(email string) (User, error) {
	for _, user := range ten.Users {
		if user.Email == email {
			return user, nil
		}
	}
	return User{}, errors.New("user not found")
}

func (ten *Tenant) HasUser(email string) bool {
	for _, user := range ten.Users {
		if user.Email == email {
			return true
		}
	}
	return false
}

func (ten *Tenant) UserIdx(email string) int {
	for i, user := range ten.Users {
		if user.Email == email {
			return i
		}
	}
	return -1
}

func (ten *Tenant) UpdateUser(email string, f func(*User) error) error {
	for i, user := range ten.Users {
		if user.Email == email {
			err := f(&ten.Users[i])
			if err != nil {
				return err
			}
			return nil
		}
	}
	return errors.New("user not found")
}

type Tenants struct {
	Entries []*Tenant `json:"entries"`
	path    string
	byID    map[uuid.UUID]*Tenant
	byName  map[string]*Tenant
}

func newTenantsPath(
	ctx context.Context, awsCfg aws.Config, stsClient AWSSTS) (string, error) {

	o, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(
		"tenants.%s.%s.json", awsCfg.Region, aws.ToString(o.Account)), nil
}

func LoadTenants(
	ctx context.Context,
	awsCfg aws.Config,
	stsClient AWSSTS,
	cogIdp AWSCognitoIdentityProvider) (*Tenants, error) {

	path, err := newTenantsPath(ctx, awsCfg, stsClient)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return InitTenants(ctx, awsCfg, stsClient, cogIdp)
		}
		return nil, err
	}
	return loadTenantsFromPath(ctx, awsCfg, stsClient, cogIdp, path)
}

func InitTenants(
	ctx context.Context,
	awsCfg aws.Config,
	stsClient AWSSTS,
	cogIdp AWSCognitoIdentityProvider) (*Tenants, error) {

	rootTenant := NewTenant()
	rootTenant.Name = RootTenantName
	rootTenant.FullName = RootTenantFullName
	rootTenant.UserPoolName = os.Getenv("LANDLORD_DEFAULT_USER_POOL")
	up, err := GetUserPoolByNameOrID(ctx, cogIdp, rootTenant.UserPoolName)
	if err != nil {
		return nil, err
	}
	rootTenant.UserPoolID = aws.ToString(up.Id)
	path, err := newTenantsPath(ctx, awsCfg, stsClient)
	if err != nil {
		return nil, err
	}
	rootTenant.CustomAuthDomain = aws.ToString(up.CustomDomain)
	result := &Tenants{
		Entries: []*Tenant{&rootTenant},
		path:    path,
		byID:    make(map[uuid.UUID]*Tenant),
		byName:  make(map[string]*Tenant),
	}
	result.byID[rootTenant.ID] = result.Entries[0]
	result.byName[rootTenant.Name] = result.Entries[0]
	return result, nil
}

func loadTenantsFromPath(
	ctx context.Context,
	awsCfg aws.Config,
	stsClient AWSSTS,
	cogIdp AWSCognitoIdentityProvider,
	path string) (*Tenants, error) {

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)
	result, err := InitTenants(ctx, awsCfg, stsClient, cogIdp)
	if err != nil {
		return nil, err
	}
	err = json.NewDecoder(file).Decode(result)
	if err != nil {
		return nil, err
	}
	for _, ten := range result.Entries {
		result.byID[ten.ID] = ten
		result.byName[ten.Name] = ten
	}
	return result, nil
}

func (tens *Tenants) Root() *Tenant {
	return tens.byName[RootTenantName]
}

func (tens *Tenants) Save() error {
	file, err := os.Create(tens.path)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)
	enc := json.NewEncoder(file)
	enc.SetIndent("  ", "  ")
	err = enc.Encode(tens)
	if err != nil {
		return err
	}
	return nil
}

func (tens *Tenants) Add(ten Tenant) error {
	if _, ok := tens.byID[ten.ID]; ok {
		return errors.New("tenant already exists")
	}
	if _, ok := tens.byName[ten.Name]; ok {
		return errors.New("tenant name already exists")
	}
	tens.Entries = append(tens.Entries, &ten)
	tens.byID[ten.ID] = tens.Entries[len(tens.Entries)-1]
	tens.byName[ten.Name] = tens.Entries[len(tens.Entries)-1]
	return tens.Save()
}

func (tens *Tenants) Delete(ten Tenant) error {
	idx := slices.IndexFunc(tens.Entries, func(t *Tenant) bool {
		return t.ID == ten.ID
	})
	if idx == -1 {
		return errors.New("tenant not found")
	}
	tens.Entries = append(tens.Entries[:idx], tens.Entries[idx+1:]...)
	delete(tens.byID, ten.ID)
	delete(tens.byName, ten.Name)
	return tens.Save()
}

func (tens *Tenants) DeleteAll() error {
	tens.Entries = []*Tenant{}
	tens.byID = make(map[uuid.UUID]*Tenant)
	tens.byName = make(map[string]*Tenant)
	return tens.Save()
}

func (tens *Tenants) GetTenantByID(id uuid.UUID) (*Tenant, error) {
	ten, ok := tens.byID[id]
	if !ok {
		return nil, errors.New("tenant not found")
	}
	return ten, nil
}

func (tens *Tenants) GetTenantByName(name string) (*Tenant, error) {
	ten, ok := tens.byName[name]
	if !ok {
		return nil, errors.New("tenant not found")
	}
	return ten, nil
}

func (tens *Tenants) ResolveTenant(h TenantHandle) (*Tenant, error) {
	if !h.IsSet() {
		return nil, errors.New("tenant ID or tenant name must be set")
	}
	if h.ID.Version() != 0 {
		return tens.GetTenantByID(h.ID)
	}
	return tens.GetTenantByName(h.Name)
}

func (tens *Tenants) UpdateTenant(id uuid.UUID, f func(*Tenant) error) error {
	ten, err := tens.GetTenantByID(id)
	if err != nil {
		return err
	}
	if err := f(ten); err != nil {
		return err
	}
	return tens.Save()
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

func (tens *Tenants) CreateTenant(
	ctx context.Context,
	iam AWSIAM,
	cogIdp AWSCognitoIdentityProvider,
	name, fullName, customAuthDomain string) (Tenant, error) {

	if err := ValidateTenantName(name); err != nil {
		return Tenant{}, err
	}
	if _, err := tens.GetTenantByName(name); err == nil {
		return Tenant{}, fmt.Errorf("tenant already exists: %s", name)
	}
	ten := NewTenant()
	ten.Name = strcase.ToKebab(name)
	ten.FullName = fullName
	ten.CustomAuthDomain = customAuthDomain
	if ten.CustomAuthDomain != "" {
		ten.UserPoolName = fmt.Sprintf("tenant-user-pool-%s", ten.Name)
		up, err := CreateUserPool(ctx, cogIdp, tens.Root().UserPoolID, ten.UserPoolName)
		if err != nil {
			return Tenant{}, err
		}
		ten.UserPoolID = aws.ToString(up.Id)
	} else {
		ten.UserPoolName = tens.Root().UserPoolName
		ten.UserPoolID = tens.Root().UserPoolID
		ten.CustomAuthDomain = tens.Root().CustomAuthDomain
	}
	role, err := CreateTenantRole(ctx, iam, ten.ID)
	if err != nil {
		return Tenant{}, err
	}
	if err := tens.Add(ten); err != nil {
		return Tenant{}, err
	}
	// TODO: create a group with the given role
	_ = role
	return ten, nil
}

func (tens *Tenants) DeleteTenant(ten *Tenant) error {
	idx := slices.IndexFunc(tens.Entries, func(t *Tenant) bool {
		return t.ID == ten.ID
	})
	if idx != -1 {
		delete(tens.byID, ten.ID)
		delete(tens.byName, ten.Name)
		tens.Entries = append(tens.Entries[:idx], tens.Entries[idx+1:]...)
		err := tens.Save()
		if err != nil {
			return err
		}
	}
	return nil
}

func (tens *Tenants) ForUsername(email string) ([]*Tenant, error) {
	var result []*Tenant
	for _, ten := range tens.Entries {
		if ten.HasUser(email) {
			result = append(result, ten)
		}
	}
	return result, nil
}

func (tens *Tenants) ForExclusiveEmail(email string) (*Tenant, error) {
	e, err := ParseEmail(email)
	if err != nil {
		return nil, err
	}
	return tens.ForExclusiveEmailDomain(e.Domain())
}

func (tens *Tenants) ForExclusiveEmailDomain(emailDomain string) (*Tenant, error) {
	var result []*Tenant
	for _, ten := range tens.Entries {
		if slices.Contains(ten.ExclusiveEmailDomains, emailDomain) {
			result = append(result, ten)
		}
	}
	if len(result) > 1 {
		return nil, errors.New("multiple tenants found")
	}
	if len(result) == 0 {
		return nil, errors.New("no tenants found")
	}
	return result[0], nil
}

func AddAllowedEmailDomain(tens *Tenants, tenID uuid.UUID, domain string) error {
	domain, err := validEmailDomain(domain)
	if err != nil {
		return err
	}
	return tens.UpdateTenant(tenID, func(t *Tenant) error {
		if slices.Contains(t.AllowedEmailDomains, domain) {
			return errors.New("domain already configured")
		}
		if _, err := tens.ForExclusiveEmailDomain(domain); err == nil {
			return errors.New("domain already configured exclusively for another tenant")
		}
		t.AllowedEmailDomains = append(t.AllowedEmailDomains, domain)
		return nil
	})
}

func AddExclusiveEmailDomain(tens *Tenants, tenID uuid.UUID, domain string) error {
	domain, err := validEmailDomain(domain)
	if err != nil {
		return err
	}
	return tens.UpdateTenant(tenID, func(t *Tenant) error {
		if belongsTo, err := tens.ForExclusiveEmailDomain(domain); err == nil {
			if t.ID == belongsTo.ID {
				return errors.New("domain already configured")
			} else {
				zap.L().Info("domain already configured for another tenant",
					zap.String("domain", domain),
					zap.String("this", t.Name),
					zap.String("another", t.Name))
				return fmt.Errorf("domain already configured for another tenant")
			}
		}
		t.ExclusiveEmailDomains = append(t.ExclusiveEmailDomains, domain)
		return nil
	})
}

func RemoveAllowedEmailDomain(tens *Tenants, tenID uuid.UUID, domain string) error {
	domain, err := validEmailDomain(domain)
	if err != nil {
		return err
	}
	return tens.UpdateTenant(tenID, func(t *Tenant) error {
		idx := slices.Index(t.AllowedEmailDomains, domain)
		if idx == -1 {
			return errors.New("domain not configured")
		}
		t.AllowedEmailDomains = append(
			t.AllowedEmailDomains[:idx], t.AllowedEmailDomains[idx+1:]...)
		return nil
	})
}

func RemoveExclusiveEmailDomain(tens *Tenants, tenID uuid.UUID, domain string) error {
	domain, err := validEmailDomain(domain)
	if err != nil {
		return err
	}
	return tens.UpdateTenant(tenID, func(t *Tenant) error {
		idx := slices.Index(t.ExclusiveEmailDomains, domain)
		if idx == -1 {
			return errors.New("domain not configured")
		}
		t.ExclusiveEmailDomains = append(
			t.ExclusiveEmailDomains[:idx], t.ExclusiveEmailDomains[idx+1:]...)
		return nil
	})
}

func ApplyTenant(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	ten *Tenant) error {

	up, err := GetUserPoolByNameOrID(ctx, cogIdp, ten.UserPoolName)
	if err != nil {
		return err
	}
	for _, user := range ten.Users {
		input := &cognitoidentityprovider.AdminUpdateUserAttributesInput{
			UserPoolId: up.Id,
			Username:   aws.String(user.Email),
			UserAttributes: []cogidptypes.AttributeType{
				{
					Name:  aws.String("custom:tenant"),
					Value: aws.String(ten.ID.String()),
				},
			},
		}
		_, err := cogIdp.AdminUpdateUserAttributes(ctx, input)
		if err != nil {
			return err
		}
	}
	return nil
}
