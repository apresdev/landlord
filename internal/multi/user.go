package multi

import (
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/apresdev/landlord/internal/apperr"
	"github.com/google/uuid"
)

type UserStatus string

const (
	UserStatusInvited  UserStatus = "invited"
	UserStatusInactive UserStatus = "inactive"
	UserStatusActive   UserStatus = "active"
)

func ParseUserStatus(s string) (UserStatus, error) {
	switch s {
	case "invited":
		return UserStatusInvited, nil
	case "inactive":
		return UserStatusInactive, nil
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

type MFAStatus string

const (
	MFAStatusNone    MFAStatus = "mfa-none"
	MFAStatusPending MFAStatus = "mfa-pending"
	MFAStatusActive  MFAStatus = "mfa-active"
)

func ParseMFAStatus(s string) (MFAStatus, error) {
	switch s {
	case "mfa-none":
		return MFAStatusNone, nil
	case "mfa-pending":
		return MFAStatusPending, nil
	case "mfa-active":
		return MFAStatusActive, nil
	default:
		zap.L().Info("invalid MFA status", zap.String("status", s))
		return MFAStatusNone, apperr.NewBadRequestE(
			fmt.Errorf("invalid MFA status: %s", s))
	}
}

func (ms MFAStatus) String() string {
	return string(ms)
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
	TenantID   uuid.UUID  `dynamodbav:"-"` // pk
	ID         uuid.UUID  `dynamodbav:"-"` // sk
	Username   string     `dynamodbav:"-"` // altName
	Role       string     `dynamodbav:"role"`
	Status     UserStatus `dynamodbav:"status"`
	MFAStatus  MFAStatus  `dynamodbav:"mfaStatus"`
	MFAPolicy  MFAPolicy  `dynamodbav:"mfaPolicy"`
	CreatedAt  string     `dynamodbav:"createdAt"`
	RawProfile []byte     `dynamodbav:"rawProfile"`
	Profile    *Profile   `dynamodbav:"-"`
}

func NewUser(tenantID uuid.UUID, cognitoID uuid.UUID, username, role string, mfaPolicy MFAPolicy) *User {
	mfaStatus := MFAStatusNone
	if mfaPolicy == MFAPolicyRequired {
		mfaStatus = MFAStatusPending
	}
	return &User{
		TenantID:  tenantID,
		ID:        cognitoID,
		Username:  username,
		Role:      role,
		Status:    UserStatusInvited,
		MFAPolicy: mfaPolicy,
		MFAStatus: mfaStatus,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

func lazyUnmarshalProfile(profile *Profile, rawProfile []byte) *Profile {
	if profile != nil {
		return profile
	}
	if rawProfile != nil {
		var result Profile
		err := json.Unmarshal(rawProfile, &result)
		if err != nil {
			zap.L().Error("error unmarshalling profile", zap.Error(err))
			return &Profile{}
		}
		return &result
	}
	return &Profile{}
}

func (u *User) ProfileField(name string) string {
	u.Profile = lazyUnmarshalProfile(u.Profile, u.RawProfile)
	f := u.Profile.GetField(name)
	if f == nil {
		return EmptyProfileValue
	}
	return f.Value
}

func (u *User) UpdateProfileField(name string, format ProfileFieldFormat, value string) {
	u.Profile = lazyUnmarshalProfile(u.Profile, u.RawProfile)
	u.Profile.SetField(name, format, value)
}

func (u *User) ProfileFields() []ProfileField {
	u.Profile = lazyUnmarshalProfile(u.Profile, u.RawProfile)
	return u.Profile.Fields
}
