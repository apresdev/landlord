package multi_test

import (
	"errors"
	"testing"

	"github.com/apresdev/landlord/internal/multi"

	"github.com/apresdev/landlord/internal/apperr"
)

func TestIsBuiltinRole(t *testing.T) {
	tests := []struct {
		name string
		role string
		want bool
	}{
		{
			name: "admin is builtin",
			role: multi.RoleAdmin,
			want: true,
		},
		{
			name: "guest is builtin",
			role: multi.RoleGuest,
			want: true,
		},
		{
			name: "custom role is not builtin",
			role: "developer",
			want: false,
		},
		{
			name: "empty role is not builtin",
			role: "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := multi.IsBuiltinRole(tt.role)
			if got != tt.want {
				t.Errorf("IsBuiltinRole() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateRoleName(t *testing.T) {
	tests := []struct {
		name       string
		roleName   string
		wantErr    bool
		errType    string
		errMessage string
	}{
		{
			name:     "valid role name",
			roleName: "developer",
			wantErr:  false,
		},
		{
			name:     "valid role name with numbers",
			roleName: "developer123",
			wantErr:  false,
		},
		{
			name:     "valid role name with underscore",
			roleName: "developer_role",
			wantErr:  false,
		},
		{
			name:     "valid role name with hyphen",
			roleName: "developer-role",
			wantErr:  false,
		},
		{
			name:       "empty role name",
			roleName:   "",
			wantErr:    true,
			errType:    "BadRequest",
			errMessage: "role name cannot be empty",
		},
		{
			name:       "too long role name",
			roleName:   "this-role-name-is-way-too-long-and-should-exceed-the-maximum-allowed-length",
			wantErr:    true,
			errType:    "BadRequest",
			errMessage: "role name must be 32 characters or less",
		},
		{
			name:       "role name with uppercase letters",
			roleName:   "Developer",
			wantErr:    true,
			errType:    "BadRequest",
			errMessage: "role name must be all lowercase and contain only letters, numbers, underscores, and hyphens",
		},
		{
			name:       "role name with spaces",
			roleName:   "developer role",
			wantErr:    true,
			errType:    "BadRequest",
			errMessage: "role name must be all lowercase and contain only letters, numbers, underscores, and hyphens",
		},
		{
			name:       "role name with special characters",
			roleName:   "developer@role",
			wantErr:    true,
			errType:    "BadRequest",
			errMessage: "role name must be all lowercase and contain only letters, numbers, underscores, and hyphens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := multi.ValidateRoleName(tt.roleName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRoleName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				// Check if the error is of the expected type and has the expected message
				if tt.errType == "BadRequest" {
					var badRequestErr apperr.BadRequest
					if !errors.As(err, &badRequestErr) {
						t.Errorf("Expected BadRequest error but got: %T", err)
					}
					if badRequestErr.Error() != tt.errMessage {
						t.Errorf("Error message = %v, want %v", badRequestErr.Error(), tt.errMessage)
					}
				}
			}
		})
	}
}

func TestRoleSlug(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "lowercase no spaces",
			input: "developer",
			want:  "developer",
		},
		{
			name:  "uppercase",
			input: "ADMIN",
			want:  "admin",
		},
		{
			name:  "mixed case with spaces",
			input: "Senior Developer",
			want:  "senior-developer",
		},
		{
			name:  "with leading and trailing spaces",
			input: "  Project Manager  ",
			want:  "project-manager",
		},
		{
			name:  "multiple spaces",
			input: "Quality  Assurance   Engineer",
			want:  "quality--assurance---engineer",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := multi.RoleSlug(tt.input); got != tt.want {
				t.Errorf("RoleSlug() = %v, want %v", got, tt.want)
			}
		})
	}
}
