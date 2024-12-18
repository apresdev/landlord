package multi

import (
	"fmt"
	"regexp"
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
		Name:   name,
		Format: format,
		Value:  value,
	})
}

const EmptyProfileValue = ""

type ProfileFieldFormat int

const (
	ProfileFieldFormatNone ProfileFieldFormat = iota
	ProfileFieldFormatName
	ProfileFieldFormatURI
	ProfileFieldFormatPhoneNumber
	ProfileFieldFormatEmail
	ProfileFieldFormatString
)

func (pff ProfileFieldFormat) String() string {
	switch pff {
	case ProfileFieldFormatName:
		return "name"
	case ProfileFieldFormatURI:
		return "uri"
	case ProfileFieldFormatPhoneNumber:
		return "phone-number"
	case ProfileFieldFormatEmail:
		return "email"
	case ProfileFieldFormatString:
		return "string"
	default:
		return "unknown"
	}
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
	default:
		return "text"
	}
}

type ProfileField struct {
	Name   string
	Format ProfileFieldFormat
	Value  string
}

type ProfileDefinition struct {
	Fields []ProfileFieldDefinition
}

func (pd *ProfileDefinition) GetFieldDefinition(name string) *ProfileFieldDefinition {
	for _, f := range pd.Fields {
		if f.Name == name {
			return &f
		}
	}
	return nil
}

type ProfileFieldDefinition struct {
	Name           string
	UIName         string
	Format         ProfileFieldFormat
	MaxLen         int
	MinLen         int
	PreventEmpty   bool
	RegexValidator string
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

var BasicProfileDefinition = ProfileDefinition{
	Fields: []ProfileFieldDefinition{
		{
			Name:   "first-name",
			UIName: "First Name",
			Format: ProfileFieldFormatName,
			MaxLen: 50,
			MinLen: 1,
		},
		{
			Name:   "last-name",
			UIName: "Last Name",
			Format: ProfileFieldFormatName,
			MaxLen: 50,
			MinLen: 1,
		},
		{
			Name:   "alternate-email",
			UIName: "Alternate Email",
			Format: ProfileFieldFormatEmail,
			MaxLen: 100,
			MinLen: 5,
		},
		{
			Name:   "phone-number",
			UIName: "Phone Number",
			Format: ProfileFieldFormatPhoneNumber,
			MaxLen: 20,
			MinLen: 1,
		},
		{
			Name:   "profile-picture",
			UIName: "Profile Picture",
			Format: ProfileFieldFormatURI,
			MaxLen: 255,
			MinLen: 5,
		},
		{
			Name:   "theme",
			UIName: "Theme",
			Format: ProfileFieldFormatString,
			MaxLen: 50,
			MinLen: 1,
		},
	},
}
