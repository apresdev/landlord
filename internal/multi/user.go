package multi

import (
	"fmt"

	"github.com/apresdev/landlord/internal/apperr"
	"github.com/google/uuid"
)

type UserStatus string

const (
	UserStatusInactive UserStatus = "inactive"
	UserStatusInvited  UserStatus = "invited"
	UserStatusActive   UserStatus = "active"
)

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
	Username  string     `dynamodbav:"username"`
	Role      string     `dynamodbav:"role"`
	Status    UserStatus `dynamodbav:"status"`
	TenantID  uuid.UUID  `dynamodbav:"tenant"`
	MFAPolicy MFAPolicy  `dynamodbav:"mfa_policy"`
}
