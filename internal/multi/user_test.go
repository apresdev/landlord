package multi_test

import (
	"testing"
	"time"

	"github.com/apresdev/landlord/internal/multi"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestParseUserStatus(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    multi.UserStatus
		expectError bool
	}{
		{"invited", "invited", multi.UserStatusInvited, false},
		{"inactive", "inactive", multi.UserStatusInactive, false},
		{"active", "active", multi.UserStatusActive, false},
		{"invalid", "invalid", multi.UserStatusInvited, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := multi.ParseUserStatus(tt.input)
			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.expected, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestUserStatusString(t *testing.T) {
	tests := []struct {
		status   multi.UserStatus
		expected string
	}{
		{multi.UserStatusInvited, "invited"},
		{multi.UserStatusInactive, "inactive"},
		{multi.UserStatusActive, "active"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.String())
		})
	}
}

func TestParseMFAStatus(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    multi.MFAStatus
		expectError bool
	}{
		{"none", "mfa-none", multi.MFAStatusNone, false},
		{"pending", "mfa-pending", multi.MFAStatusPending, false},
		{"active", "mfa-active", multi.MFAStatusActive, false},
		{"invalid", "invalid", multi.MFAStatusNone, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := multi.ParseMFAStatus(tt.input)
			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.expected, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestMFAStatusString(t *testing.T) {
	tests := []struct {
		status   multi.MFAStatus
		expected string
	}{
		{multi.MFAStatusNone, "mfa-none"},
		{multi.MFAStatusPending, "mfa-pending"},
		{multi.MFAStatusActive, "mfa-active"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.String())
		})
	}
}

func TestParseMFAPolicy(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    multi.MFAPolicy
		expectError bool
	}{
		{"none", "none", multi.MFAPolicyNone, false},
		{"required", "required", multi.MFAPolicyRequired, false},
		{"invalid", "invalid", multi.MFAPolicyNone, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := multi.ParseMFAPolicy(tt.input)
			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.expected, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestMFAPolicyString(t *testing.T) {
	tests := []struct {
		policy   multi.MFAPolicy
		expected string
	}{
		{multi.MFAPolicyNone, "none"},
		{multi.MFAPolicyRequired, "required"},
		{multi.MFAPolicy(99), "unknown"}, // Testing default case
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.policy.String())
		})
	}
}

func TestMFAPolicyStrings(t *testing.T) {
	expected := []string{"none", "required"}
	assert.Equal(t, expected, multi.MFAPolicyStrings())
}

func TestNewUser(t *testing.T) {
	tenantID := uuid.New()
	cognitoID := uuid.New()
	username := "testuser"
	role := "admin"

	t.Run("with policy none", func(t *testing.T) {
		user := multi.NewUser(tenantID, cognitoID, username, role, multi.MFAPolicyNone)

		assert.Equal(t, tenantID, user.TenantID)
		assert.NotEqual(t, uuid.Nil, user.ID)
		assert.Equal(t, username, user.Username)
		assert.Equal(t, cognitoID, user.CognitoID)
		assert.Equal(t, role, user.Role)
		assert.Equal(t, multi.UserStatusInvited, user.Status)
		assert.Equal(t, multi.MFAPolicyNone, user.MFAPolicy)
		assert.Equal(t, multi.MFAStatusNone, user.MFAStatus)

		// Validate CreatedAt is in correct format
		_, err := time.Parse(time.RFC3339, user.CreatedAt)
		assert.NoError(t, err)
	})

	t.Run("with policy required", func(t *testing.T) {
		user := multi.NewUser(tenantID, cognitoID, username, role, multi.MFAPolicyRequired)

		assert.Equal(t, tenantID, user.TenantID)
		assert.Equal(t, multi.MFAPolicyRequired, user.MFAPolicy)
		assert.Equal(t, multi.MFAStatusPending, user.MFAStatus)
	})
}
