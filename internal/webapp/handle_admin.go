package webapp

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"slices"
	"sort"

	"github.com/apresdev/landlord/internal/apperr"
	"github.com/apresdev/landlord/internal/authx"
	"github.com/apresdev/landlord/internal/db"
	"github.com/apresdev/landlord/internal/jwx"
	"github.com/apresdev/landlord/internal/multi"
	"github.com/apresdev/landlord/internal/rest"
	"github.com/apresdev/landlord/internal/tinytools"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/google/uuid"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
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

func handleAdminDashboard(a App, c *gin.Context) error {
	cu, err := resolveCurrentUser(a, c)
	if err != nil {
		clearLandlordCookies(c, a.Settings().URL())
		redirectToLogin(c)
		return nil
	}
	stats := newDashboardStats()
	return render(c, dashboardPage("Landlord", cu, stats))
}

func handleAdminDashboardError(a App, c *gin.Context) error {
	errMsg := c.Query("error")
	if errMsg == "" {
		errMsg = "Unknown error"
	} else {
		if decodedErrMsg, err := base64.URLEncoding.DecodeString(errMsg); err != nil {
			errMsg = "Decoding error"
		} else {
			errMsg = string(decodedErrMsg)
		}
	}
	err := render(c, errorPage(
		a.Settings().AppName(),
		a.Settings().AdminEmail(),
		mailToAdminLink(a.Settings().AdminEmail()),
		errMsg,
		"Return to dashboard",
		templ.URL(appPath("/admin/dashboard"))))

	if err != nil {
		// We are already in an error context and have hit another error.
		// Log it and just try redirecting to the root page. Something is
		// really wrong if we get here.
		zap.L().Error("Error rendering error page", zap.Error(err))
		c.Redirect(http.StatusFound, appPath("/"))
	}

	return nil
}

func handleAdminTenants(a App, c *gin.Context) error {
	cu, err := resolveCurrentUser(a, c)
	if err != nil {
		clearLandlordCookies(c, a.Settings().URL())
		redirectToLogin(c)
		return nil
	}

	// TODO: all calls to GetAllTenants and similar APIs need to check permissions.

	tens, err := db.GetAllTenants(c.Request.Context(), a.DDB())
	if err != nil {
		return err
	}
	return render(c, tenantsPage("Landlord", cu, tens))
}

func handleAdminTenantSelect(a App, c *gin.Context) error {
	cu, err := resolveCurrentUser(a, c)
	if err != nil {
		clearLandlordCookies(c, a.Settings().URL())
		redirectToLogin(c)
		return nil
	}
	action := c.Query("action")
	if action == "" {
		return apperr.NewBadRequest("missing action")
	}
	tens, err := db.GetAllTenants(c.Request.Context(), a.DDB())
	if err != nil {
		return err
	}
	rc, err := authx.RequesterFromContext(c)
	if err != nil {
		clearLandlordCookies(c, a.Settings().URL())
		redirectToLogin(c)
		return err
	}
	authorizedTenants := make([]*multi.Tenant, 0)
	for _, ten := range tens {
		if authx.IsAuthorizedForTenant(a.RootTenantID(), ten, rc.Roles) {
			authorizedTenants = append(authorizedTenants, ten)
		}
	}

	return render(c, tenantSelectPage("Landlord", cu, tens))
}

func handleAdminNewTenant(a App, c *gin.Context) error {
	return render(c, newTenantPage(a.Settings().AppName()))
}

func handleAdminNewTenantSubmit(a App, c *gin.Context) error {
	var req rest.CreateTenantRequest
	req.Name = c.PostForm("name")
	req.Description = c.PostForm("description")
	_, err := rest.HandleCreateTenant(a, c, req)
	if err != nil {
		return err
	}
	redirectTo(c, "admin/tenants")
	return nil
}

func handleAdminActivateTenantSubmit(a App, c *gin.Context) error {
	i := c.Param("id")
	if i == "" {
		return fmt.Errorf("no id provided")
	}
	ten, err := rest.HandleSetTenantState(a, c, i, string(multi.TenantStateActive))
	if err != nil {
		return err
	}
	return render(c, tenantFragment(ten))
}

func handleAdminDeactivateTenantSubmit(a App, c *gin.Context) error {
	i := c.Param("id")
	if i == "" {
		return fmt.Errorf("no id provided")
	}
	ten, err := rest.HandleSetTenantState(a, c, i, string(multi.TenantStateInactive))
	if err != nil {
		return err
	}
	return render(c, tenantFragment(ten))
}

type RoleWithCount struct {
	Role             string
	Count            int
	RemovableTooltip string
}

func rolesWithCount(roles []string, users []*multi.User) []RoleWithCount {
	sort.Strings(roles)
	roleCounts := make(map[string]int)
	for _, u := range users {
		roleCounts[u.Role]++
	}
	var rwc []RoleWithCount
	for _, r := range roles {
		tooltip := ""
		if multi.IsBuiltinRole(r) {
			tooltip = "Cannot remove built-in role"
		} else if roleCounts[r] > 0 {
			tooltip = "Cannot remove role that is in use"
		}
		rwc = append(rwc, RoleWithCount{
			Role:             r,
			Count:            roleCounts[r],
			RemovableTooltip: tooltip,
		})
	}
	sort.Slice(rwc, func(i, j int) bool {
		return rwc[i].Role < rwc[j].Role
	})
	return rwc
}

func handleAdminTenant(a App, c *gin.Context) error {
	cu, err := resolveCurrentUser(a, c)
	if err != nil {
		clearLandlordCookies(c, a.Settings().URL())
		redirectToLogin(c)
		return nil
	}

	id := c.Param("tenantID")
	th, err := multi.ParseTenantHandle(id)
	if err != nil {
		return apperr.NewBadRequestE(
			fmt.Errorf("error parsing tenant ID: %w", err))
	}
	ten, err := db.ResolveTenant(c.Request.Context(), a.DDB(), th)
	if err != nil && !apperr.IsNotFound(err) {
		return apperr.NewNotFoundE(
			fmt.Errorf("error resolving tenant: %w", err))
	}
	if err != nil {
		return apperr.NewInternalE(
			fmt.Errorf("error getting tenants: %w", err))
	}

	return render(c, tenantPage(a.Settings().AppName(), cu, ten))
}

func handleAdminTenantTab(a App, c *gin.Context) error {
	cu, err := resolveCurrentUser(a, c)
	if err != nil {
		clearLandlordCookies(c, a.Settings().URL())
		redirectToLogin(c)
		return nil
	}

	id := c.Param("tenantID")
	th, err := multi.ParseTenantHandle(id)
	if err != nil {
		return apperr.NewBadRequestE(
			fmt.Errorf("error parsing tenant ID: %w", err))
	}
	ten, err := db.ResolveTenant(c.Request.Context(), a.DDB(), th)
	if err != nil && !apperr.IsNotFound(err) {
		return apperr.NewNotFoundE(
			fmt.Errorf("error resolving tenant: %w", err))
	}
	if err != nil {
		return apperr.NewInternalE(
			fmt.Errorf("error getting tenants: %w", err))
	}

	// TODO: handle no users for tenant more gracefully

	users, err := db.GetUsersForTenant(c.Request.Context(), a.DDB(), ten.ID)
	if err != nil {
		return apperr.NewInternalE(
			fmt.Errorf("error getting users for tenant: %w", err))
	}
	sort.Slice(users, func(i, j int) bool {
		return users[i].Username < users[j].Username
	})
	tab := c.Param("tab")
	if tab == "summary" {
		cognitoURL := authx.CognitoUserPoolURL(a.Cfg().Region, ten.UserPoolID)
		return render(c, tenantSummaryTab(
			tab, a.Settings().AppName(), cu, ten, cognitoURL, users, rolesWithCount(ten.Roles, users)))
	} else if tab == "users" {
		return render(c, tenantUsersTab(
			tab, a.Settings().AppName(), cu, ten, users, rolesWithCount(ten.Roles, users)))
	} else if tab == "roles" {
		return render(c, tenantRolesTab(
			tab, a.Settings().AppName(), cu, ten, users, rolesWithCount(ten.Roles, users)))
	} else {
		zap.L().Info("unknown tenant tab", zap.String("tab", tab))
	}
	return nil
}

func handleAdminAddRoleSubmit(a App, c *gin.Context) error {
	role := c.PostForm("role")
	tenantID := c.Param("tenantID")
	th, err := multi.ParseTenantHandle(tenantID)
	if err != nil {
		return apperr.NewBadRequestE(err)
	}
	ten, err := db.ResolveTenant(c.Request.Context(), a.DDB(), th)
	if err != nil {
		return apperr.NewNotFoundE(err)
	}
	if slices.Contains(ten.Roles, role) {
		users, err := db.GetUsersForTenant(c.Request.Context(), a.DDB(), ten.ID)
		if err != nil {
			return apperr.NewInternalE(err)
		}
		return render(c, tenantAddRoleTableAndForm(ten, rolesWithCount(ten.Roles, users),
			"Role already exists: "+role))
	} else {
		var req rest.TenantRolesRequest
		req.Roles = append(ten.Roles, role)
		req.SubTenantRoles = ten.SubTenantRoles
		err = rest.HandleSetRolesForTenant(a, c, req, tenantID)
		if err != nil {
			return apperr.NewInternalE(err)
		}
		users, err := db.GetUsersForTenant(c.Request.Context(), a.DDB(), ten.ID)
		if err != nil {
			return apperr.NewInternalE(err)
		}
		zap.L().Info("Adding role to tenant", zap.String("role", role))
		return render(c, tenantAddRoleTableAndForm(ten, rolesWithCount(req.Roles, users), ""))
	}
}

func handleAdminDeleteRoleSubmit(a App, c *gin.Context) error {
	role := c.Param("role")
	tenantID := c.Param("tenantID")
	th, err := multi.ParseTenantHandle(tenantID)
	if err != nil {
		return apperr.NewBadRequestE(err)
	}
	ten, err := db.ResolveTenant(c.Request.Context(), a.DDB(), th)
	if err != nil {
		return apperr.NewNotFoundE(err)
	}
	if !slices.Contains(ten.Roles, role) {
		return apperr.NewBadRequestE(fmt.Errorf("role does not exist: %s", role))
	}
	var req rest.TenantRolesRequest
	req.Roles = tinytools.RemoveFromSlice(ten.Roles, role)
	req.SubTenantRoles = ten.SubTenantRoles
	err = rest.HandleSetRolesForTenant(a, c, req, tenantID)
	if err != nil {
		return apperr.NewInternalE(err)
	}
	users, err := db.GetUsersForTenant(c.Request.Context(), a.DDB(), ten.ID)
	if err != nil {
		return apperr.NewInternalE(err)
	}
	zap.L().Info("Removed role to tenant", zap.String("role", role))
	return render(c, tenantAddRoleTableAndForm(ten, rolesWithCount(req.Roles, users), ""))
}

func handleAdminInviteUserForm(a App, c *gin.Context) error {
	tenantID := c.Param("tenantID")
	th, err := multi.ParseTenantHandle(tenantID)
	if err != nil {
		return apperr.NewBadRequestE(err)
	}
	ten, err := db.ResolveTenant(c.Request.Context(), a.DDB(), th)
	if err != nil {
		return apperr.NewNotFoundE(err)
	}
	return render(c, tenantInviteUserForm(tenantID, ten, multi.MFAPolicyStrings()))
}

func handleAdminInviteUserButton(a App, c *gin.Context) error {
	return render(c, tenantInviteUserButton(c.Param("tenantID")))
}

func handleAdminInviteUserSubmit(a App, c *gin.Context) error {
	tenantID, err := rest.UUIDParam(c, "tenantID")
	if err != nil {
		return err
	}
	var req rest.InviteRequest
	req.Username = c.PostForm("username")
	req.Role = c.PostForm("role")
	req.MfaPolicy = rest.MfaPolicy(c.PostForm("mfaPolicy"))
	if err := rest.HandleInviteToTenant(a, c, tenantID, req); err != nil {
		return err
	}
	users, err := db.GetUsersForTenant(c.Request.Context(), a.DDB(), tenantID)
	if err != nil {
		return apperr.NewInternalE(err)
	}
	return render(c, tenantInviteUserTableAndForm(tenantID, users, ""))
}

func handleAdminAddRoleForm(a App, c *gin.Context) error {
	return render(c, tenantAddRoleForm(c.Param("tenantID")))
}

func handleAdminAddRoleButton(a App, c *gin.Context) error {
	return render(c, tenantAddRoleButton(c.Param("tenantID")))
}

func handleAdminActivateUserSubmit(a App, c *gin.Context) error {
	tenantID := c.Param("tenantID")
	if tenantID == "" {
		return apperr.NewBadRequest("no tenant ID provided")
	}
	uID := c.Param("userID")
	if uID == "" {
		return apperr.NewBadRequest("no user ID provided")
	}
	userID, err := uuid.Parse(uID)
	if err != nil {
		return apperr.NewBadRequestE(
			fmt.Errorf("error parsing user ID: %w", err))
	}
	user, err := rest.HandleSetUserStatus(
		a, c, tenantID, userID, string(multi.TenantStateActive))
	if err != nil {
		return err
	}
	return render(c, userFragment(user))
}

func handleAdminDeactivateUserSubmit(a App, c *gin.Context) error {
	tenantID := c.Param("tenantID")
	if tenantID == "" {
		return apperr.NewBadRequest("no tenant ID provided")
	}
	uID := c.Param("userID")
	if uID == "" {
		return apperr.NewBadRequest("no user ID provided")
	}
	userID, err := uuid.Parse(uID)
	if err != nil {
		return apperr.NewBadRequestE(
			fmt.Errorf("error parsing user ID: %w", err))
	}
	user, err := rest.HandleSetUserStatus(
		a, c, tenantID, userID, string(multi.TenantStateInactive))
	if err != nil {
		return err
	}
	return render(c, userFragment(user))
}

func handleAdminCancelInviteSubmit(a App, c *gin.Context) error {
	tenantID, err := rest.UUIDParam(c, "tenantID")
	if err != nil {
		return err
	}
	userID, err := rest.UUIDParam(c, "userID")
	if err != nil {
		return err
	}
	if err := rest.HandleCancelInvitation(a, c, tenantID, userID); err != nil {
		return err
	}
	c.String(http.StatusOK, "<div></div>")
	return nil
}

func handleAdminUpdateMFAPolicy(a App, c *gin.Context, policy rest.MfaPolicy) error {
	tenantID, err := rest.UUIDParam(c, "tenantID")
	if err != nil {
		return err
	}
	userID, err := rest.UUIDParam(c, "userID")
	if err != nil {
		return err
	}
	req := rest.UpdateUserMfaRequest{}
	req.MfaPolicy = policy
	if err := rest.HandleUpdateUserMFA(a, c, tenantID, userID, req); err != nil {
		return err
	}
	user, err := db.GetUserInTenantByID(c.Request.Context(), a.DDB(), tenantID, userID)
	if err != nil {
		return apperr.NewInternalE(err)
	}
	return render(c, userFragment(user))
}

func handleAdminEnableMFASubmit(a App, c *gin.Context) error {
	return handleAdminUpdateMFAPolicy(a, c, rest.MFAPolicyRequired)
}

func handleAdminDisableMFASubmit(a App, c *gin.Context) error {
	return handleAdminUpdateMFAPolicy(a, c, rest.MFAPolicyNone)
}

func handleAdminUser(a App, c *gin.Context) error {
	cu, err := resolveCurrentUser(a, c)
	if err != nil {
		clearLandlordCookies(c, a.Settings().URL())
		redirectToLogin(c)
		return nil
	}

	tenantID := c.Param("tenantID")
	th, err := multi.ParseTenantHandle(tenantID)
	if err != nil {
		return apperr.NewBadRequestE(
			fmt.Errorf("error parsing tenant ID: %w", err))
	}
	ten, err := db.ResolveTenant(c.Request.Context(), a.DDB(), th)
	if err != nil && !apperr.IsNotFound(err) {
		return apperr.NewNotFoundE(
			fmt.Errorf("error resolving tenant: %w", err))
	}
	if err != nil {
		return apperr.NewInternalE(
			fmt.Errorf("error getting tenants: %w", err))
	}

	userID := c.Param("userID")
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return apperr.NewBadRequestE(
			fmt.Errorf("error parsing user ID: %w", err))
	}
	u, err := db.GetUserInTenantByID(c.Request.Context(), a.DDB(), ten.ID, userUUID)
	if err != nil {
		return apperr.NewNotFoundE(
			fmt.Errorf("error getting user: %w", err))
	}

	cogU, err := a.CogIdp().AdminGetUser(c.Request.Context(), &cognitoidentityprovider.AdminGetUserInput{
		UserPoolId: aws.String(ten.UserPoolID),
		Username:   aws.String(u.Username),
	})
	if err != nil {
		return apperr.NewInternalE(
			fmt.Errorf("error getting user from Cognito: %w", err))
	}
	_ = cogU

	cogURL := authx.CognitoUserDetailsURL(a.Cfg().Region, ten.UserPoolID, u.ID.String())
	return render(c, userSummary(a.Settings().AppName(), cu, u, cogURL, ten))
}

func handleAdminProfile(a App, c *gin.Context) error {
	var rawTokens jwx.RawTokens
	rawTokens.IdToken, _ = c.Cookie(LandlordIDTokenCookie)
	rawTokens.AccessToken, _ = c.Cookie(LandlordAccessTokenCookie)
	rawTokens.RefreshToken, _ = c.Cookie(LandlordRefreshTokenCookie)
	if rawTokens.IdToken == "" || rawTokens.AccessToken == "" || rawTokens.RefreshToken == "" {
		redirectToLogin(c)
		return nil
	}
	username, _ := c.Cookie(LandlordUsernameCookie)
	if username == "" {
		redirectToLogin(c)
		return nil
	}
	_, tenant, err := db.GetUserAndTenant(c.Request.Context(), a.DDB(), username)
	if err != nil {
		zap.L().Warn("error finding tenant for username",
			zap.String("username", username),
			zap.Error(err))
		clearLandlordCookies(c, a.Settings().URL())
		redirectToLogin(c)
		return nil
	}
	validationURL := jwx.ValidationURL(a.Cfg().Region, tenant.UserPoolID)
	tks, err := jwx.ValidateAllTokens(
		c.Request.Context(), clientID(c), a.Validator(), validationURL, rawTokens)
	if err != nil {
		redirectToLogin(c)
		return err
	}

	idTokDesc, err := jwx.Describe("ID Token", tks.IdToken)
	if err != nil {
		return err
	}
	accessTokDesc, err := jwx.Describe("Access Token", tks.AccessToken)
	if err != nil {
		return err
	}
	return render(c, displayJWTPage(idTokDesc, accessTokDesc))
}
