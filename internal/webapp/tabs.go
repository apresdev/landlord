package webapp

import (
	"fmt"

	"go.uber.org/zap"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type TenantTab string

const (
	tenantSummaryTab  TenantTab = "summary"
	tenantUsersTab    TenantTab = "users"
	tenantInvitesTab  TenantTab = "invites"
	tenantRolesTab    TenantTab = "roles"
	tenantSubRolesTab TenantTab = "subroles"
)

var tenantTabs = []TenantTab{
	tenantSummaryTab,
	tenantUsersTab,
	tenantInvitesTab,
	tenantRolesTab,
	tenantSubRolesTab,
}

func newTenantTab(s string) TenantTab {
	if s == "" {
		return tenantSummaryTab
	}
	switch s {
	case "summary":
		return tenantSummaryTab
	case "users":
		return tenantUsersTab
	case "invites":
		return tenantInvitesTab
	case "roles":
		return tenantRolesTab
	case "subroles":
		return tenantSubRolesTab
	default:
		zap.L().Info("bad value for tenant tab", zap.String("tab", s))
		return tenantSummaryTab
	}
}

func parseTabFromAction(action string) TenantTab {
	switch action {
	case "manage-users":
		return tenantUsersTab
	case "manage-invites":
		return tenantInvitesTab
	case "manage-roles":
		return tenantRolesTab
	default:
		zap.L().Error("bad value for tab action", zap.String("tab", action))
	}
	return tenantSummaryTab
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
