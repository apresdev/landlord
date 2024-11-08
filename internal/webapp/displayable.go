package webapp

import (
	"time"

	"github.com/apresdev/landlord/internal/multi"
)

func displayTimestamp(t string) string {
	parsedTime, err := time.Parse(time.RFC3339, t)
	if err != nil {
		return t
	}
	return parsedTime.Format("2006-01-02 15:04:05") + " UTC"
}

type DisplayableUser struct {
	User *multi.User
}

func NewDisplayableUser(user *multi.User) *DisplayableUser {
	return &DisplayableUser{User: user}
}

func NewDisplayableUsers(users []*multi.User) []*DisplayableUser {
	dus := make([]*DisplayableUser, len(users))
	for i, user := range users {
		dus[i] = NewDisplayableUser(user)
	}
	return dus
}

func (du *DisplayableUser) MFAStatus() string {
	switch du.User.MFAStatus {
	case multi.MFAStatusNone:
		return "No MFA required"
	case multi.MFAStatusPending:
		return "Pending MFA setup"
	case multi.MFAStatusActive:
		return "Active"
	default:
		return "Unknown"
	}
}

func (du *DisplayableUser) Status() string {
	switch du.User.Status {
	case multi.UserStatusInvited:
		return "Invited"
	case multi.UserStatusInactive:
		return "Inactive"
	case multi.UserStatusActive:
		return "Active"
	default:
		return "Unknown"
	}
}
