package multi

import (
	"reflect"
	"strings"
	"testing"
)

func TestProfileGetField(t *testing.T) {
	// Setup test profile
	profile := Profile{
		Fields: []ProfileField{
			{
				Name:      "name",
				NewFormat: ProfileFieldFormatName,
				Value:     "John Doe",
			},
			{
				Name:      "email",
				NewFormat: ProfileFieldFormatEmail,
				Value:     "john@example.com",
			},
		},
	}

	// Test cases
	tests := []struct {
		name      string
		fieldName string
		want      *ProfileField
		wantNil   bool
	}{
		{
			name:      "existing field - name",
			fieldName: "name",
			want: &ProfileField{
				Name:      "name",
				NewFormat: ProfileFieldFormatName,
				Value:     "John Doe",
			},
		},
		{
			name:      "existing field - email",
			fieldName: "email",
			want: &ProfileField{
				Name:      "email",
				NewFormat: ProfileFieldFormatEmail,
				Value:     "john@example.com",
			},
		},
		{
			name:      "non-existing field",
			fieldName: "phone",
			wantNil:   true,
		},
	}

	// Run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := profile.GetField(tt.fieldName)
			if tt.wantNil {
				if got != nil {
					t.Errorf("Profile.GetField() = %v, want nil", got)
				}
				return
			}
			if !reflect.DeepEqual(*got, *tt.want) {
				t.Errorf("Profile.GetField() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProfileSetField(t *testing.T) {
	// Test cases
	tests := []struct {
		name        string
		initial     Profile
		fieldName   string
		fieldFormat ProfileFieldFormat
		fieldValue  string
		want        Profile
	}{
		{
			name: "update existing field",
			initial: Profile{
				Fields: []ProfileField{
					{
						Name:      "name",
						NewFormat: ProfileFieldFormatName,
						Value:     "John Doe",
					},
				},
			},
			fieldName:   "name",
			fieldFormat: ProfileFieldFormatName,
			fieldValue:  "Jane Doe",
			want: Profile{
				Fields: []ProfileField{
					{
						Name:      "name",
						NewFormat: ProfileFieldFormatName,
						Value:     "Jane Doe",
					},
				},
			},
		},
		{
			name: "add new field",
			initial: Profile{
				Fields: []ProfileField{
					{
						Name:      "name",
						NewFormat: ProfileFieldFormatName,
						Value:     "John Doe",
					},
				},
			},
			fieldName:   "email",
			fieldFormat: ProfileFieldFormatEmail,
			fieldValue:  "john@example.com",
			want: Profile{
				Fields: []ProfileField{
					{
						Name:      "name",
						NewFormat: ProfileFieldFormatName,
						Value:     "John Doe",
					},
					{
						Name:      "email",
						NewFormat: ProfileFieldFormatEmail,
						Value:     "john@example.com",
					},
				},
			},
		},
		{
			name:        "add to empty profile",
			initial:     Profile{},
			fieldName:   "name",
			fieldFormat: ProfileFieldFormatName,
			fieldValue:  "John Doe",
			want: Profile{
				Fields: []ProfileField{
					{
						Name:      "name",
						NewFormat: ProfileFieldFormatName,
						Value:     "John Doe",
					},
				},
			},
		},
	}

	// Run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := tt.initial
			profile.SetField(tt.fieldName, tt.fieldFormat, tt.fieldValue)
			if !reflect.DeepEqual(profile, tt.want) {
				t.Errorf("Profile.SetField() resulted in %v, want %v", profile, tt.want)
			}
		})
	}
}

func TestProfileFieldFormatString(t *testing.T) {
	tests := []struct {
		format ProfileFieldFormat
		want   string
	}{
		{ProfileFieldFormatName, "name"},
		{ProfileFieldFormatURI, "uri"},
		{ProfileFieldFormatPhoneNumber, "phone-number"},
		{ProfileFieldFormatEmail, "email"},
		{ProfileFieldFormatString, "string"},
		{ProfileFieldFormatSelect, "select"},
		{ProfileFieldFormat("custom"), "custom"},
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			if got := tt.format.String(); got != tt.want {
				t.Errorf("ProfileFieldFormat.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProfileFieldFormatHTMLInputType(t *testing.T) {
	tests := []struct {
		format ProfileFieldFormat
		want   string
	}{
		{ProfileFieldFormatName, "text"},
		{ProfileFieldFormatURI, "url"},
		{ProfileFieldFormatPhoneNumber, "tel"},
		{ProfileFieldFormatEmail, "email"},
		{ProfileFieldFormatString, "text"},
		{ProfileFieldFormatSelect, "select"},
		{ProfileFieldFormat("custom"), "text"}, // Default case
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			if got := tt.format.HTMLInputType(); got != tt.want {
				t.Errorf("ProfileFieldFormat.HTMLInputType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProfileDefinitionGetFieldDefinition(t *testing.T) {
	definition := ProfileDefinition{
		Fields: []ProfileFieldDefinition{
			{
				Name:   "name",
				UIName: "Full Name",
				Format: ProfileFieldFormatName,
			},
			{
				Name:   "email",
				UIName: "Email Address",
				Format: ProfileFieldFormatEmail,
			},
		},
	}

	tests := []struct {
		name      string
		fieldName string
		want      *ProfileFieldDefinition
		wantNil   bool
	}{
		{
			name:      "existing field - name",
			fieldName: "name",
			want: &ProfileFieldDefinition{
				Name:   "name",
				UIName: "Full Name",
				Format: ProfileFieldFormatName,
			},
		},
		{
			name:      "existing field - email",
			fieldName: "email",
			want: &ProfileFieldDefinition{
				Name:   "email",
				UIName: "Email Address",
				Format: ProfileFieldFormatEmail,
			},
		},
		{
			name:      "non-existing field",
			fieldName: "phone",
			wantNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := definition.GetFieldDefinition(tt.fieldName)
			if tt.wantNil {
				if got != nil {
					t.Errorf("ProfileDefinition.GetFieldDefinition() = %v, want nil", got)
				}
				return
			}
			if !reflect.DeepEqual(*got, *tt.want) {
				t.Errorf("ProfileDefinition.GetFieldDefinition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProfileFieldDefinitionValidateValue(t *testing.T) {
	tests := []struct {
		name        string
		definition  ProfileFieldDefinition
		value       string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid value",
			definition: ProfileFieldDefinition{
				Name:         "name",
				UIName:       "Full Name",
				Format:       ProfileFieldFormatName,
				MaxLen:       100,
				MinLen:       2,
				PreventEmpty: true,
			},
			value:   "John Doe",
			wantErr: false,
		},
		{
			name: "empty value with prevent empty",
			definition: ProfileFieldDefinition{
				Name:         "name",
				UIName:       "Full Name",
				Format:       ProfileFieldFormatName,
				PreventEmpty: true,
			},
			value:       "",
			wantErr:     true,
			errContains: "cannot be empty",
		},
		{
			name: "too long value",
			definition: ProfileFieldDefinition{
				Name:   "name",
				UIName: "Full Name",
				Format: ProfileFieldFormatName,
				MaxLen: 5,
			},
			value:       "John Doe",
			wantErr:     true,
			errContains: "too long",
		},
		{
			name: "too short value",
			definition: ProfileFieldDefinition{
				Name:   "name",
				UIName: "Full Name",
				Format: ProfileFieldFormatName,
				MinLen: 10,
				MaxLen: 100,
			},
			value:       "John",
			wantErr:     true,
			errContains: "too short",
		},
		{
			name: "valid regex",
			definition: ProfileFieldDefinition{
				Name:           "email",
				UIName:         "Email Address",
				Format:         ProfileFieldFormatEmail,
				RegexValidator: "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$",
				MaxLen:         100,
			},
			value:   "john@example.com",
			wantErr: false,
		},
		{
			name: "invalid regex",
			definition: ProfileFieldDefinition{
				Name:           "email",
				UIName:         "Email Address",
				Format:         ProfileFieldFormatEmail,
				RegexValidator: "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$",
				MaxLen:         100,
			},
			value:       "invalid-email",
			wantErr:     true,
			errContains: "does not match regex",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.definition.ValidateValue(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ProfileFieldDefinition.ValidateValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error message '%v' does not contain '%v'", err.Error(), tt.errContains)
				}
			}
		})
	}
}
