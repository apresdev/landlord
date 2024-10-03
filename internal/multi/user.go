package multi

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/apresdev/landlord/internal/apperr"
	"github.com/google/uuid"
)

type UserStatus string

const (
	UserStatusInvited    UserStatus = "invited"
	UserStatusInactive   UserStatus = "inactive"
	UserStatusPendingMFA UserStatus = "pending-mfa"
	UserStatusActive     UserStatus = "active"
)

func ParseUserStatus(s string) (UserStatus, error) {
	switch s {
	case "invited":
		return UserStatusInvited, nil
	case "inactive":
		return UserStatusInactive, nil
	case "pending-mfa":
		return UserStatusPendingMFA, nil
	case "active":
		return UserStatusActive, nil
	default:
		zap.L().Info("invalid user status", zap.String("status", s))
		return UserStatusInvited, apperr.NewBadRequestE(
			fmt.Errorf("invalid user status: %s", s))
	}
}

func (u UserStatus) String() string {
	return string(u)
}

// MFAPolicy is the policy for multi-factor authentication. Note that this
// value is serialized to the database as an integer -- do not re-order.
type MFAPolicy int

// We do not use iota here because we want to control the serialization of
// this type to the database.
const (
	MFAPolicyNone     MFAPolicy = 0
	MFAPolicyRequired MFAPolicy = 1
)

func ParseMFAPolicy(s string) (MFAPolicy, error) {
	switch s {
	case "none":
		return MFAPolicyNone, nil
	case "required":
		return MFAPolicyRequired, nil
	default:
		return MFAPolicyNone, apperr.NewBadRequestE(
			fmt.Errorf("invalid mfa policy: %s", s))
	}
}

func MFAPolicyStrings() []string {
	return []string{MFAPolicyNone.String(), MFAPolicyRequired.String()}
}

func (m MFAPolicy) String() string {
	switch m {
	case MFAPolicyNone:
		return "none"
	case MFAPolicyRequired:
		return "required"
	default:
		return "unknown"
	}
}

type User struct {
	TenantID  uuid.UUID  `dynamodbav:"-"` // pk
	ID        uuid.UUID  `dynamodbav:"-"` // sk
	Username  string     `dynamodbav:"-"` // altName
	Role      string     `dynamodbav:"role"`
	Status    UserStatus `dynamodbav:"status"`
	MFAPolicy MFAPolicy  `dynamodbav:"mfaPolicy"`
}
