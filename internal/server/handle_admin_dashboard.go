package server

import (
	"github.com/gin-gonic/gin"
)

type DashboardStats struct {
	OutstandingInvites    string
	ConfiguredTenantCount string
	ConfiguredUserCount   string
	SecurityAlertMessage  string
}

func newDashboardStats() DashboardStats {
	// TODO: real stats
	return DashboardStats{
		OutstandingInvites:    "12",
		ConfiguredTenantCount: "4",
		ConfiguredUserCount:   "32",
		SecurityAlertMessage:  "No current security alerts",
	}
}

func handleAdminDashboard(a *app, c *gin.Context) error {
	cUser, err := resolveClientUser(a, c)
	if err != nil {
		clearLandlordCookies(c, a.Settings().URL())
		redirectToLogin(c)
		return nil
	}
	stats := newDashboardStats()
	render(c, dashboardPage("Landlord", cUser.User, stats))
	return nil
}
