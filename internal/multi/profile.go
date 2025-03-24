package multi

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"go.uber.org/zap"
)

type Profile struct {
	Fields []ProfileField
}

func (p *Profile) GetField(name string) *ProfileField {
	for _, f := range p.Fields {
		if f.Name == name {
			return &f
		}
	}
	return nil
}

func (p *Profile) SetField(name string, format ProfileFieldFormat, value string) {
	for i, f := range p.Fields {
		if f.Name == name {
			p.Fields[i].Value = value
			return
		}
	}
	p.Fields = append(p.Fields, ProfileField{
		Name:      name,
		NewFormat: format,
		Value:     value,
	})
}

const EmptyProfileValue = ""

type ProfileFieldFormat string

const (
	ProfileFieldFormatName        = ProfileFieldFormat("name")
	ProfileFieldFormatURI         = ProfileFieldFormat("uri")
	ProfileFieldFormatPhoneNumber = ProfileFieldFormat("phone-number")
	ProfileFieldFormatEmail       = ProfileFieldFormat("email")
	ProfileFieldFormatString      = ProfileFieldFormat("string")
	ProfileFieldFormatSelect      = ProfileFieldFormat("select")
)

func (pff ProfileFieldFormat) String() string {
	return string(pff)
}

func (pff ProfileFieldFormat) HTMLInputType() string {
	switch pff {
	case ProfileFieldFormatName:
		return "text"
	case ProfileFieldFormatURI:
		return "url"
	case ProfileFieldFormatPhoneNumber:
		return "tel"
	case ProfileFieldFormatEmail:
		return "email"
	case ProfileFieldFormatString:
		return "text"
	case ProfileFieldFormatSelect:
		return "select"
	default:
		return "text"
	}
}

type ProfileField struct {
	Name string
	// Called NewFormat as the serialization changed from a number to a string
	NewFormat ProfileFieldFormat
	Value     string
}

type ProfileDefinition struct {
	Fields []ProfileFieldDefinition `json:"fields"`
}

func (pd ProfileDefinition) GetFieldDefinition(name string) *ProfileFieldDefinition {
	for _, f := range pd.Fields {
		if f.Name == name {
			return &f
		}
	}
	return nil
}

type ProfileFieldDefinition struct {
	Name                 string             `json:"name"`
	UIName               string             `json:"ui_name"`
	Format               ProfileFieldFormat `json:"format"`
	MaxLen               int                `json:"max_len,omitempty"`
	MinLen               int                `json:"min_len,omitempty"`
	PreventEmpty         bool               `json:"prevent_empty,omitempty"`
	RegexValidator       string             `json:"regex_validator,omitempty"`
	InviteTimeField      bool               `json:"invite_time_field,omitempty"`
	PossibleValuesURL    string             `json:"possible_values_url,omitempty"`
	PossibleValuesJQExpr string             `json:"possible_values_jq_expr,omitempty"`
}

func (pfd *ProfileFieldDefinition) ValidateValue(value string) error {
	if pfd.PreventEmpty && value == "" {
		return fmt.Errorf("field %s cannot be empty", pfd.Name)
	}
	if len(value) > pfd.MaxLen {
		return fmt.Errorf("field %s is too long", pfd.Name)
	}
	if value != "" && len(value) < pfd.MinLen {
		return fmt.Errorf("field %s is too short", pfd.Name)
	}
	if pfd.RegexValidator != "" {
		matched, err := regexp.MatchString(pfd.RegexValidator, value)
		if err != nil {
			return fmt.Errorf("error validating field %s", pfd.Name)
		}
		if !matched {
			return fmt.Errorf("field %s does not match regex", pfd.Name)
		}
	}
	return nil
}

var userProfileDefinition ProfileDefinition
var tenantProfileDefinition ProfileDefinition

func init() {
	userProfEnv := os.Getenv("LANDLORD_USER_PROFILE_DEFINITION")
	if userProfEnv != "" {
		prof, err := parseProfileDefinition(userProfEnv)
		if err != nil {
			zap.L().Error("failed to parse user profile definition", zap.Error(err))
		} else {
			userProfileDefinition = prof
		}
	}
	tenantProfEnv := os.Getenv("LANDLORD_TENANT_PROFILE_DEFINITION")
	if tenantProfEnv != "" {
		prof, err := parseProfileDefinition(tenantProfEnv)
		if err != nil {
			zap.L().Error("failed to parse tenant profile definition", zap.Error(err))
		} else {
			tenantProfileDefinition = prof
		}
	}
}

// parseProfileDefinition parses s as a base64-encoded JSON encoding of a ProfileDefinition.
func parseProfileDefinition(s string) (ProfileDefinition, error) {
	decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(s))
	jsonBytes, err := io.ReadAll(decoder)
	if err != nil {
		return ProfileDefinition{}, err
	}
	var def ProfileDefinition
	err = json.Unmarshal(jsonBytes, &def)
	if err != nil {
		return ProfileDefinition{}, err
	}
	return def, nil
}

func GetUserProfileDefinition() ProfileDefinition {
	return userProfileDefinition
}

func GetTenantProfileDefinition() ProfileDefinition {
	return tenantProfileDefinition
}

type ProfileFieldProvider interface {
	ProfileFieldValue(name string) string
	ProfileField(name string) *ProfileField
	UpdateProfileField(name string, format ProfileFieldFormat, value string)
	ProfileFields() []ProfileField
}

type TenantProfileDefinitions struct {
	TenantScope ProfileDefinition
	UserScope   ProfileDefinition
}
