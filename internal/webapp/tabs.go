package webapp

import (
	"fmt"

	"go.uber.org/zap"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type TenantTab string

const (
	tenantSummaryTab TenantTab = "summary"
	tenantProfileTab TenantTab = "profile"
	tenantUsersTab   TenantTab = "users"
	tenantInvitesTab TenantTab = "invites"
	tenantRolesTab   TenantTab = "roles"
)

var tenantTabs = []TenantTab{
	tenantSummaryTab,
	tenantProfileTab,
	tenantUsersTab,
	tenantInvitesTab,
	tenantRolesTab,
}

func newTenantTab(s string) TenantTab {
	if s == "" {
		return tenantSummaryTab
	}
	switch s {
	case "summary":
		return tenantSummaryTab
	case "profile":
		return tenantProfileTab
	case "users":
		return tenantUsersTab
	case "invites":
		return tenantInvitesTab
	case "roles":
		return tenantRolesTab
	default:
		zap.L().Info("bad value for tenant tab", zap.String("tab", s))
		return tenantSummaryTab
	}
}

func (tt TenantTab) TabTemplateName() string {
	return fmt.Sprintf("tenant%sTab",
		cases.Title(language.English).String(string(tt)))
}

type profileTab string

const (
	profileTabSummary     profileTab = "summary"
	profileTabDetails     profileTab = "details"
	profileTabAccessToken profileTab = "access-token"
)

var profileTabs = []profileTab{profileTabSummary, profileTabDetails, profileTabAccessToken}

func newProfileTab(s string) profileTab {
	if s == "" {
		return profileTabSummary
	}
	switch s {
	case "summary":
		return profileTabSummary
	case "details":
		return profileTabDetails
	case "access-token":
		return profileTabAccessToken
	default:
		zap.L().Info("bad value for profile tab", zap.String("tab", s))
		return profileTabSummary
	}
}

func (pt profileTab) title() string {
	switch pt {
	case profileTabSummary:
		return "Summary"
	case profileTabDetails:
		return "Profile"
	case profileTabAccessToken:
		return "Access Token"
	}
	return ""
}

type UserTab string

const (
	userSummaryTab UserTab = "summary"
	userProfileTab UserTab = "profile"
)

var userTabs = []UserTab{
	userSummaryTab,
	userProfileTab,
}

func newUserTab(s string) UserTab {
	if s == "" {
		return userSummaryTab
	}
	switch s {
	case "summary":
		return userSummaryTab
	case "profile":
		return userProfileTab
	default:
		zap.L().Info("bad value for tenant tab", zap.String("tab", s))
		return userSummaryTab
	}
}

func (ut UserTab) TabTemplateName() string {
	return fmt.Sprintf("user%sTab",
		cases.Title(language.English).String(string(ut)))
}
