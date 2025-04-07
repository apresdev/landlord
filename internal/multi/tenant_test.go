package multi_test

import (
	"testing"

	"github.com/apresdev/landlord/internal/multi"

	"github.com/google/uuid"
)

func TestValidateTenantName(t *testing.T) {
	tests := []struct {
		name       string
		tenantName string
		wantErr    bool
		errMessage string
	}{
		{
			name:       "valid name",
			tenantName: "acme-corp",
			wantErr:    false,
		},
		{
			name:       "valid name with numbers",
			tenantName: "company123",
			wantErr:    false,
		},
		{
			name:       "empty name",
			tenantName: "",
			wantErr:    true,
			errMessage: "tenant name cannot be empty",
		},
		{
			name:       "too long name",
			tenantName: "this-tenant-name-is-way-too-long-and-exceeds-the-maximum-allowed-length",
			wantErr:    true,
			errMessage: "tenant name must be 32 characters or less",
		},
		{
			name:       "name with uppercase letters",
			tenantName: "CompanyName",
			wantErr:    true,
			errMessage: "tenant name must start with a letter",
		},
		{
			name:       "name with spaces",
			tenantName: "acme corp",
			wantErr:    true,
			errMessage: "tenant name must be all lowercase and contain only letters, numbers, and hyphens",
		},
		{
			name:       "name with special characters",
			tenantName: "acme@corp",
			wantErr:    true,
			errMessage: "tenant name must be all lowercase and contain only letters, numbers, and hyphens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := multi.ValidateTenantName(tt.tenantName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTenantName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errMessage != "" {
				if err.Error() != tt.errMessage {
					t.Errorf("ValidateTenantName() error message = %v, want %v", err.Error(), tt.errMessage)
				}
			}
		})
	}
}

func TestIsRootTenant(t *testing.T) {
	tests := []struct {
		name     string
		tenantID uuid.UUID
		want     bool
	}{
		{
			name:     multi.RootTenantName,
			tenantID: uuid.New(),
			want:     true,
		},
		{
			name:     "not-root",
			tenantID: uuid.New(),
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ten := multi.NewTenant(tt.name)
			if got := ten.IsRoot(); got != tt.want {
				t.Errorf("IsRoot() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateTenantDescription(t *testing.T) {
	tests := []struct {
		name        string
		description string
		wantErr     bool
		errMessage  string
	}{
		{
			name:        "valid description",
			description: "This is a valid tenant description",
			wantErr:     false,
		},
		{
			name:        "empty description",
			description: "",
			wantErr:     false, // Assuming empty descriptions are allowed
		},
		{
			name:        "too long description",
			description: string(make([]rune, 257)), // Assuming max length is 256
			wantErr:     true,
			errMessage:  "description must be 256 characters or less",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := multi.ValidateTenantDescription(tt.description)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTenantDescription() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errMessage != "" {
				if err.Error() != tt.errMessage {
					t.Errorf("ValidateTenantDescription() error message = %v, want %v", err.Error(), tt.errMessage)
				}
			}
		})
	}
}
