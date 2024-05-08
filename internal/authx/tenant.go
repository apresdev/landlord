package authx

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"slices"

	"github.com/google/uuid"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"

	"github.com/ettle/strcase"
)

type TenantID string

func MustNewTenantID() TenantID {
	return TenantID(uuid.New().String())
}

type Tenant struct {
	ID           TenantID `json:"id"`
	Name         string   `json:"name"`
	FullName     string   `json:"full_name"`
	EmailDomains []string `json:"email_domains"`
	UserEmails   []string `json:"users"`
	UserPool     string   `json:"user_pool"`
}

func NewTenant() Tenant {
	return Tenant{
		ID: MustNewTenantID(),
	}
}

type Tenants struct {
	Entries []Tenant `json:"entries"`
	path    string
	byID    map[TenantID]Tenant
	byName  map[string]Tenant
}

const defaultTenantsPath = "tenants.json"

func newTenants() Tenants {
	return Tenants{
		path:   defaultTenantsPath,
		byID:   make(map[TenantID]Tenant),
		byName: make(map[string]Tenant),
	}
}

func LoadTenants(path string) (Tenants, error) {
	file, err := os.Open(path)
	if err != nil {
		return Tenants{}, err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)
	tens := newTenants()
	tens.path = path
	err = json.NewDecoder(file).Decode(&tens)
	if err != nil {
		return Tenants{}, err
	}
	for _, ten := range tens.Entries {
		tens.byID[ten.ID] = ten
		tens.byName[ten.Name] = ten
	}
	return tens, nil
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
	tens.Entries = append(tens.Entries, ten)
	tens.byID[ten.ID] = ten
	tens.byName[ten.Name] = ten
	return tens.Save()
}

func (tens *Tenants) Replace(ten Tenant) error {
	idx := slices.IndexFunc(tens.Entries, func(tenant Tenant) bool {
		return tenant.ID == ten.ID
	})
	if idx == -1 {
		return errors.New("tenant not found")
	}
	tens.Entries[idx] = ten
	tens.byID[ten.ID] = ten
	tens.byName[ten.Name] = ten
	return tens.Save()
}

var AllTenants Tenants

func init() {
	var err error
	_, err = os.Stat(defaultTenantsPath)
	if err != nil {
		AllTenants = newTenants()
	} else {
		AllTenants, err = LoadTenants(defaultTenantsPath)
		if err != nil {
			panic(err)
		}
	}
}

func GetTenantByID(id TenantID) (Tenant, error) {
	ten, ok := AllTenants.byID[id]
	if !ok {
		return Tenant{}, errors.New("tenant not found")
	}
	return ten, nil
}

func GetTenantByName(name string) (Tenant, error) {
	ten, ok := AllTenants.byName[name]
	if !ok {
		return Tenant{}, errors.New("tenant not found")
	}
	return ten, nil
}

func ResolveTenant(tenantID TenantID, tenantName string) (Tenant, error) {
	if tenantID == "" && tenantName == "" {
		return Tenant{}, errors.New("tenant ID or tenant name must be set")
	}
	if tenantID != "" {
		return GetTenantByID(tenantID)
	}
	return GetTenantByName(tenantName)
}

const maxTenantNameLength = 32

func CreateTenant(userPool, name string) (Tenant, error) {
	if name == "" {
		return Tenant{}, errors.New("tenant name must be set")
	}
	if len(name) > maxTenantNameLength {
		return Tenant{}, errors.New("tenant name must be 32 characters or less")
	}
	ten := NewTenant()
	ten.Name = strcase.ToKebab(name)
	ten.UserPool = userPool
	if err := AllTenants.Add(ten); err != nil {
		return Tenant{}, err
	}
	return ten, nil
}

func DeleteTenant(ten Tenant) error {
	idx := slices.IndexFunc(AllTenants.Entries, func(tenant Tenant) bool {
		return tenant.ID == ten.ID
	})
	if idx != -1 {
		delete(AllTenants.byID, ten.ID)
		delete(AllTenants.byName, ten.Name)
		AllTenants.Entries = append(AllTenants.Entries[:idx], AllTenants.Entries[idx+1:]...)
		err := AllTenants.Save()
		if err != nil {
			return err
		}
	}
	return nil
}

func AddUserToTenant(ten Tenant, email string) error {
	idx := slices.IndexFunc(ten.UserEmails, func(email string) bool {
		return email == email
	})
	if idx != -1 {
		return errors.New("user already in tenant")
	}
	ten.UserEmails = append(ten.UserEmails, email)
	return AllTenants.Replace(ten)
}

func ApplyTenant(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	ten Tenant) error {

	up, err := GetUserPoolByNameOrID(ctx, cogIdp, ten.UserPool)
	if err != nil {
		return err
	}
	for _, email := range ten.UserEmails {
		input := &cognitoidentityprovider.AdminUpdateUserAttributesInput{
			UserPoolId: up.Id,
			Username:   aws.String(email),
			UserAttributes: []types.AttributeType{
				{
					Name:  aws.String("custom:tenant"),
					Value: aws.String(string(ten.ID)),
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
