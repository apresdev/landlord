package webapp

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"slices"
	"sort"

	"github.com/apresdev/landlord/internal/tinytools"

	"github.com/apresdev/landlord/internal/perms"

	"github.com/apresdev/landlord/internal/apperr"
	"github.com/apresdev/landlord/internal/authx"
	"github.com/apresdev/landlord/internal/db"
	"github.com/apresdev/landlord/internal/multi"
	"github.com/apresdev/landlord/internal/rest"
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
	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		clearLandlordCookies(c, a.Settings().ConsoleURL())
		redirectToLogin(c)
		return nil
	}
	stats := newDashboardStats()
	ds := NewDisplayableSubject(c.Request.Context(), a, sub)
	return render(c, dashboardPage("Landlord", ds, stats))
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

func handleAdminClearImpersonation(a App, c *gin.Context) error {
	var req rest.ClearImpersonationRequest
	req.RefreshToken, _ = c.Cookie(LandlordRefreshTokenCookie)
	err := rest.HandleClearImpersonation(a, c, req)
	if err != nil {
		return err
	}
	clearLandlordCookies(c, a.Settings().ConsoleURL())
	c.Header("HX-Redirect", appPath("/"))
	c.Status(http.StatusNoContent)
	return nil
}

func handleAdminTenants(a App, c *gin.Context) error {
	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		clearLandlordCookies(c, a.Settings().ConsoleURL())
		redirectToLogin(c)
		return nil
	}
	tens, err := rest.HandleGetTenants(a, c)
	if err != nil {
		return err
	}
	ds := NewDisplayableSubject(c.Request.Context(), a, sub)
	return render(c, tenantsPage("Landlord", ds, tens))
}

type tenantTab string

const (
	tenantTabSummary  tenantTab = "summary"
	tenantTabUsers    tenantTab = "users"
	tenantTabInvites  tenantTab = "invites"
	tenantTabRoles    tenantTab = "roles"
	tenantTabSubRoles tenantTab = "subroles"
)

var tenantTabs = []tenantTab{
	tenantTabSummary,
	tenantTabUsers,
	tenantTabInvites,
	tenantTabRoles,
	tenantTabSubRoles,
}

func newTenantTab(s string) tenantTab {
	if s == "" {
		return tenantTabSummary
	}
	switch s {
	case "summary":
		return tenantTabSummary
	case "users":
		return tenantTabUsers
	case "invites":
		return tenantTabInvites
	case "roles":
		return tenantTabRoles
	case "subroles":
		return tenantTabSubRoles
	default:
		zap.L().Info("bad value for tenant tab", zap.String("tab", s))
		return tenantTabSummary
	}
}

func handleAdminTenantSelect(a App, c *gin.Context) error {
	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		clearLandlordCookies(c, a.Settings().ConsoleURL())
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
	authorizedTenants := make([]*multi.Tenant, 0)
	for _, ten := range tens {
		if perms.IsAuthorizedToReadTenant(a.RootTenantID(), ten, sub.Role) {
			authorizedTenants = append(authorizedTenants, ten)
		}
	}
	tab := tenantTabSummary
	if action == "manage-users" {
		tab = tenantTabUsers
	} else if action == "manage-roles" {
		tab = tenantTabRoles
	} else if action == "manage-invites" {
		tab = tenantTabInvites
	}
	ds := NewDisplayableSubject(c.Request.Context(), a, sub)
	return render(c, tenantSelectPage(
		a.Settings().AppName(), ds, authorizedTenants, string(tab)))
}

func handleAdminCreateTenantButton(a App, c *gin.Context) error {
	return render(c, tenantCreateTenantButton())
}

func handleAdminCreateTenantForm(a App, c *gin.Context) error {
	return render(c, tenantCreateTenantForm(""))
}

func handleAdminCreateTenantSubmit(a App, c *gin.Context) error {
	var req rest.CreateTenantRequest
	req.Name = c.PostForm("name")
	req.Description = c.PostForm("description")
	_, err := rest.HandleCreateTenant(a, c, req)
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	tens, err := rest.HandleGetTenants(a, c)
	if err != nil {
		return err
	}
	return render(c, tenantCreateTenantTableAndForm(tens, errMsg))
}

func handleAdminActivateTenantSubmit(a App, c *gin.Context) error {
	i := c.Param("tenantID")
	if i == "" {
		return fmt.Errorf("no id provided")
	}
	ten, err := rest.HandleUpdateTenantState(a, c, i, string(multi.TenantStateActive))
	if err != nil {
		return err
	}
	return render(c, tenantFragment(ten))
}

func handleAdminDeactivateTenantSubmit(a App, c *gin.Context) error {
	i := c.Param("tenantID")
	if i == "" {
		return fmt.Errorf("no id provided")
	}
	ten, err := rest.HandleUpdateTenantState(a, c, i, string(multi.TenantStateInactive))
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
	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		clearLandlordCookies(c, a.Settings().ConsoleURL())
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
	startingTab := newTenantTab(c.Query("tab"))
	ds := NewDisplayableSubject(c.Request.Context(), a, sub)
	return render(c, tenantPage(a.Settings().AppName(), ds, ten, string(startingTab)))
}

func handleAdminTenantTab(a App, c *gin.Context) error {
	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		clearLandlordCookies(c, a.Settings().ConsoleURL())
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
	users, err := db.GetUsersForTenant(c.Request.Context(), a.DDB(), ten.ID)
	if err != nil {
		return apperr.NewInternalE(
			fmt.Errorf("error getting users for tenant: %w", err))
	}
	sort.Slice(users, func(i, j int) bool {
		return users[i].Username < users[j].Username
	})
	tab := newTenantTab(c.Param("tab"))
	switch tab {
	case tenantTabSummary:
		cognitoURL := authx.CognitoUserPoolURL(a.Cfg().Region, ten.UserPoolID)
		ds := NewDisplayableSubject(c.Request.Context(), a, sub)
		baseURL := appPath("/admin/tenants/" + id)
		return render(c, tenantSummaryTab(a.Settings().AppName(), ds, ten, cognitoURL, users, baseURL))
	case tenantTabUsers:
		dus, err := NewDisplayableUsers(c.Request.Context(), a, sub, ten, users)
		if err != nil {
			return apperr.NewInternalE(err)
		}
		ds := NewDisplayableSubject(c.Request.Context(), a, sub)
		return render(c, tenantUsersTab(a.Settings().AppName(), ds, ten, dus))
	case tenantTabInvites:
		invites := make([]*multi.User, 0, len(users))
		for _, user := range users {
			if user.Status == multi.UserStatusInvited {
				invites = append(invites, user)
			}
		}
		dus, err := NewDisplayableUsers(c.Request.Context(), a, sub, ten, invites)
		if err != nil {
			return apperr.NewInternalE(err)
		}
		ds := NewDisplayableSubject(c.Request.Context(), a, sub)
		return render(c, tenantInvitesTab(a.Settings().AppName(), ds, ten, dus))
	case tenantTabRoles:
		ds := NewDisplayableSubject(c.Request.Context(), a, sub)
		return render(c, tenantRolesTab(
			a.Settings().AppName(), ds, ten, users, rolesWithCount(ten.Roles, users)))
	case tenantTabSubRoles:
		ds := NewDisplayableSubject(c.Request.Context(), a, sub)
		return render(c, tenantSubRolesTab(
			a.Settings().AppName(), ds, ten, users, rolesWithCount(ten.SubTenantRoles, users)))
	default:
		zap.L().Info("unknown tenant tab", zap.String("tab", string(tab)))
	}
	return nil
}

func handleAdminDescriptionSubmit(a App, c *gin.Context) error {
	tenantID := c.Param("tenantID")
	description := c.PostForm("description")
	req := rest.TenantSetDescriptionRequest{
		Description: description,
	}
	ten, err := rest.HandleUpdateTenantDescription(a, c, tenantID, req)
	if err != nil {
		return apperr.NewInternalE(err)
	}
	return render(c, tenantDescriptionForm(tenantID, ten.Description))
}

func handleAdminDescriptionEditForm(a App, c *gin.Context) error {
	tenantID := c.Param("tenantID")
	th, err := multi.ParseTenantHandle(tenantID)
	if err != nil {
		return apperr.NewBadRequestE(err)
	}
	ten, err := db.ResolveTenant(c.Request.Context(), a.DDB(), th)
	if err != nil {
		return apperr.NewNotFoundE(err)
	}
	return render(c, tenantDescriptionEditForm(tenantID, ten.Description))
}

func handleAdminDescriptionForm(a App, c *gin.Context) error {
	tenantID := c.Param("tenantID")
	th, err := multi.ParseTenantHandle(tenantID)
	if err != nil {
		return apperr.NewBadRequestE(err)
	}
	ten, err := db.ResolveTenant(c.Request.Context(), a.DDB(), th)
	if err != nil {
		return apperr.NewNotFoundE(err)
	}
	return render(c, tenantDescriptionForm(tenantID, ten.Description))
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
	profileFieldDefs := ten.GetUserProfileDefinition()
	return render(c, tenantInviteUserForm(
		tenantID, ten, multi.MFAPolicyStrings(), profileFieldDefs))
}

func handleAdminInviteUserButton(a App, c *gin.Context) error {
	return render(c, tenantInviteUserButton(c.Param("tenantID")))
}

func handleAdminInviteUserSubmit(a App, c *gin.Context) error {
	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		return err
	}

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

	ten, err := db.GetTenantByID(c.Request.Context(), a.DDB(), tenantID)
	if err != nil {
		return apperr.NewInternalE(err)
	}
	type remembrance struct {
		def       multi.ProfileFieldDefinition
		formValue string
	}
	remebrances := make([]remembrance, 0)
	for _, profileFieldDef := range ten.GetUserProfileDefinition().Fields {
		v := c.PostForm(profileFieldDef.Name)
		if v != "" {
			remebrances = append(remebrances, remembrance{
				def: profileFieldDef, formValue: v})
		}
	}
	if len(remebrances) > 0 {
		var updateProfileFieldsReq rest.UpdateProfileFieldsRequest
		for _, r := range remebrances {
			updateProfileFieldsReq.ProfileFields = append(
				updateProfileFieldsReq.ProfileFields,
				rest.ProfileField{
					Name:  r.def.Name,
					Value: r.formValue})
		}
		user, err := db.GetUserByName(c.Request.Context(), a.DDB(), req.Username)
		if err != nil {
			return apperr.NewNotFoundE(err)
		}
		err = rest.HandleUpdateUserProfileFields(
			a, c, tenantID, user.ID, updateProfileFieldsReq)
		if err != nil {
			return err
		}
	}

	users, err := db.GetUsersForTenant(c.Request.Context(), a.DDB(), tenantID)
	if err != nil {
		return apperr.NewInternalE(err)
	}
	ds := NewDisplayableSubject(c.Request.Context(), a, sub)
	dus, err := NewDisplayableUsers(c.Request.Context(), a, sub, ten, users)
	if err != nil {
		return apperr.NewInternalE(err)
	}
	return render(c, tenantInviteUserTableAndForm(tenantID, ds, dus, ""))
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
		err = rest.HandleUpdateRolesForTenant(a, c, req, tenantID)
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
	err = rest.HandleUpdateRolesForTenant(a, c, req, tenantID)
	if err != nil {
		return apperr.NewInternalE(err)
	}
	users, err := db.GetUsersForTenant(c.Request.Context(), a.DDB(), ten.ID)
	if err != nil {
		return apperr.NewInternalE(err)
	}
	zap.L().Info("Removed role from tenant", zap.String("role", role))
	return render(c, tenantAddRoleTableAndForm(
		ten, rolesWithCount(req.Roles, users), ""))
}

func handleAdminAddRoleForm(a App, c *gin.Context) error {
	return render(c, tenantAddRoleForm(c.Param("tenantID")))
}

func handleAdminAddRoleButton(a App, c *gin.Context) error {
	return render(c, tenantAddRoleButton(c.Param("tenantID")))
}

func handleAdminAddSubRoleSubmit(a App, c *gin.Context) error {
	role := c.PostForm("subrole")
	tenantID := c.Param("tenantID")
	th, err := multi.ParseTenantHandle(tenantID)
	if err != nil {
		return apperr.NewBadRequestE(err)
	}
	ten, err := db.ResolveTenant(c.Request.Context(), a.DDB(), th)
	if err != nil {
		return apperr.NewNotFoundE(err)
	}
	if slices.Contains(ten.SubTenantRoles, role) {
		return render(c, tenantAddSubRoleTableAndForm(
			ten, rolesWithCount(ten.SubTenantRoles, nil), "Sub-role already exists: "+role))
	} else {
		var req rest.TenantRolesRequest
		req.Roles = ten.Roles
		req.SubTenantRoles = append(ten.SubTenantRoles, role)
		err = rest.HandleUpdateRolesForTenant(a, c, req, tenantID)
		if err != nil {
			return apperr.NewInternalE(err)
		}
		zap.L().Info("Adding sub-role to tenant", zap.String("sub-role", role))
		return render(c, tenantAddSubRoleTableAndForm(
			ten, rolesWithCount(req.SubTenantRoles, nil), ""))
	}
}

func handleAdminDeleteSubRoleSubmit(a App, c *gin.Context) error {
	subRole := c.Param("subrole")
	tenantID := c.Param("tenantID")
	th, err := multi.ParseTenantHandle(tenantID)
	if err != nil {
		return apperr.NewBadRequestE(err)
	}
	ten, err := db.ResolveTenant(c.Request.Context(), a.DDB(), th)
	if err != nil {
		return apperr.NewNotFoundE(err)
	}
	if !slices.Contains(ten.SubTenantRoles, subRole) {
		return apperr.NewBadRequestE(fmt.Errorf("sub-role does not exist: %s", subRole))
	}
	var req rest.TenantRolesRequest
	req.Roles = ten.Roles
	req.SubTenantRoles = tinytools.RemoveFromSlice(ten.SubTenantRoles, subRole)
	err = rest.HandleUpdateRolesForTenant(a, c, req, tenantID)
	if err != nil {
		return apperr.NewInternalE(err)
	}
	zap.L().Info("Removed sub-role from tenant", zap.String("sub-role", subRole))
	return render(c, tenantAddSubRoleTableAndForm(
		ten, rolesWithCount(req.SubTenantRoles, nil), ""))
}

func handleAdminAddSubRoleForm(a App, c *gin.Context) error {
	return render(c, tenantAddSubRoleForm(c.Param("tenantID")))
}

func handleAdminAddSubRoleButton(a App, c *gin.Context) error {
	return render(c, tenantAddSubRoleButton(c.Param("tenantID")))
}

func handleAdminActivateUserSubmit(a App, c *gin.Context) error {
	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		return err
	}
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
	user, err := rest.HandleUpdateUserStatus(
		a, c, tenantID, userID, string(multi.TenantStateActive))
	if err != nil {
		return err
	}
	ten, err := db.GetTenantByID(c.Request.Context(), a.DDB(), user.TenantID)
	if err != nil {
		return apperr.NewInternalE(err)
	}
	ds := NewDisplayableSubject(c.Request.Context(), a, sub)
	du, err := NewDisplayableUser(c.Request.Context(), a, sub, ten, user)
	if err != nil {
		return apperr.NewInternalE(err)
	}
	return render(c, userFragment(ds, du))
}

func handleAdminDeactivateUserSubmit(a App, c *gin.Context) error {
	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		return err
	}
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
	user, err := rest.HandleUpdateUserStatus(
		a, c, tenantID, userID, string(multi.TenantStateInactive))
	if err != nil {
		return err
	}
	ten, err := db.GetTenantByID(c.Request.Context(), a.DDB(), user.TenantID)
	if err != nil {
		return apperr.NewInternalE(err)
	}
	ds := NewDisplayableSubject(c.Request.Context(), a, sub)
	du, err := NewDisplayableUser(c.Request.Context(), a, sub, ten, user)
	if err != nil {
		return apperr.NewInternalE(err)
	}
	return render(c, userFragment(ds, du))
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
	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		return err
	}
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
	ten, err := db.GetTenantByID(c.Request.Context(), a.DDB(), tenantID)
	if err != nil {
		return apperr.NewInternalE(err)
	}
	ds := NewDisplayableSubject(c.Request.Context(), a, sub)
	du, err := NewDisplayableUser(c.Request.Context(), a, sub, ten, user)
	if err != nil {
		return apperr.NewInternalE(err)
	}
	return render(c, userFragment(ds, du))
}

func handleAdminEnableMFASubmit(a App, c *gin.Context) error {
	return handleAdminUpdateMFAPolicy(a, c, rest.MFAPolicyRequired)
}

func handleAdminDisableMFASubmit(a App, c *gin.Context) error {
	return handleAdminUpdateMFAPolicy(a, c, rest.MFAPolicyNone)
}

func handleAdminReInviteUserSubmit(a App, c *gin.Context) error {
	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		return err
	}
	tenantID, err := rest.UUIDParam(c, "tenantID")
	if err != nil {
		return err
	}
	userID, err := rest.UUIDParam(c, "userID")
	if err != nil {
		return err
	}
	user, err := db.GetUserInTenantByID(c.Request.Context(), a.DDB(), tenantID, userID)
	if err != nil {
		return err
	}
	var req rest.InviteRequest
	req.Username = user.Username
	req.Role = user.Role
	req.MfaPolicy = rest.MfaPolicy(user.MFAPolicy.String())
	if err := rest.HandleInviteToTenant(a, c, tenantID, req); err != nil {
		return err
	}
	ten, err := db.GetTenantByID(c.Request.Context(), a.DDB(), tenantID)
	if err != nil {
		return apperr.NewInternalE(err)
	}
	ds := NewDisplayableSubject(c.Request.Context(), a, sub)
	du, err := NewDisplayableUser(c.Request.Context(), a, sub, ten, user)
	if err != nil {
		return apperr.NewInternalE(err)
	}
	return render(c, userFragment(ds, du))
}

func handleAdminImpersonateUserSubmit(a App, c *gin.Context) error {
	_, err := perms.SubjectFromContext(c)
	if err != nil {
		return err
	}
	tenantID, err := rest.UUIDParam(c, "tenantID")
	if err != nil {
		return err
	}
	userID, err := rest.UUIDParam(c, "userID")
	if err != nil {
		return err
	}
	user, err := db.GetUserInTenantByID(c.Request.Context(), a.DDB(), tenantID, userID)
	if err != nil {
		return err
	}
	var req rest.ImpersonateUserRequest
	req.Tenant = tenantID.String()
	req.Username = user.Username
	req.RefreshToken, _ = c.Cookie(LandlordRefreshTokenCookie)
	_, err = rest.HandleImpersonateUser(a, c, req)
	if err != nil {
		return err
	}
	clearLandlordCookies(c, a.Settings().ConsoleURL())
	c.Header("HX-Redirect", appPath("/"))
	c.Status(http.StatusNoContent)
	return nil
}

func handleAdminUser(a App, c *gin.Context) error {
	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		clearLandlordCookies(c, a.Settings().ConsoleURL())
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
	ds := NewDisplayableSubject(c.Request.Context(), a, sub)
	cogUser, err := NewDisplayableCognitoUser(
		c.Request.Context(), a, ten.UserPoolID, u.ID.String())
	if err != nil {
		return apperr.NewInternalE(
			fmt.Errorf("error getting cognito user: %w", err))
	}
	baseURL := appPath("/admin/tenants/" + tenantID + "/users/" + userID)
	return render(c, userSummary(a.Settings().AppName(), ds, ten, u, cogUser, baseURL))
}

func handleAdminProfile(a App, c *gin.Context) error {
	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		clearLandlordCookies(c, a.Settings().ConsoleURL())
		redirectToLogin(c)
		return nil
	}
	ds := NewDisplayableSubject(c.Request.Context(), a, sub)
	return render(c, profilePage(a.Settings().AppName(), ds, string(profileTabSummary)))
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

func handleAdminProfileTab(a App, c *gin.Context) error {
	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		clearLandlordCookies(c, a.Settings().ConsoleURL())
		redirectToLogin(c)
		return nil
	}

	tab := newProfileTab(c.Param("tab"))
	switch tab {
	case profileTabSummary:
		subjectTenant, err := sub.ResolveTenant(c.Request.Context(), a)
		if err != nil {
			return err
		}
		subjectUser, err := sub.ResolveUser(c.Request.Context(), a)
		if err != nil {
			return err
		}
		cog, err := NewDisplayableCognitoUser(
			c.Request.Context(), a, subjectTenant.UserPoolID, subjectUser.ID.String())
		if err != nil {
			return err
		}
		return render(c, profileSummaryTab(subjectTenant, subjectUser, cog))
	case profileTabDetails:
		subjectTenant, err := sub.ResolveTenant(c.Request.Context(), a)
		if err != nil {
			return err
		}
		subjectUser, err := sub.ResolveUser(c.Request.Context(), a)
		if err != nil {
			return err
		}
		cog, err := NewDisplayableCognitoUser(
			c.Request.Context(), a, subjectTenant.UserPoolID, subjectUser.ID.String())
		if err != nil {
			return err
		}
		return render(c, profileSummaryTab(subjectTenant, subjectUser, cog))
	case profileTabAccessToken:
		ds := NewDisplayableSubject(c.Request.Context(), a, sub)
		dt := NewDisplayableJWT("Access Token", sub.AccessToken)
		return render(c, profileTokenTab(profileTabAccessToken, ds, dt))
	default:
		zap.L().Info("unknown profile tab", zap.String("tab", string(tab)))
	}
	return nil
}

func handleAdminUserProfileFieldSubmit(a App, c *gin.Context) error {
	tenantID, err := rest.UUIDParam(c, "tenantID")
	if err != nil {
		return err
	}
	userID, err := rest.UUIDParam(c, "userID")
	if err != nil {
		return err
	}
	fieldName := c.Query("fieldName")
	fieldValue := c.PostForm("fieldValue")
	req := rest.UpdateProfileFieldsRequest{}
	req.ProfileFields = append(req.ProfileFields, rest.ProfileField{
		Name:  fieldName,
		Value: fieldValue,
	})
	err = rest.HandleUpdateUserProfileFields(a, c, tenantID, userID, req)
	if err != nil {
		return apperr.NewInternalE(err)
	}
	ten, err := db.GetTenantByID(c.Request.Context(), a.DDB(), tenantID)
	if err != nil {
		return apperr.NewInternalE(err)
	}
	defn := ten.GetUserProfileDefinition().GetFieldDefinition(fieldName)
	if defn == nil {
		return apperr.NewBadRequest("invalid field name: " + fieldName)
	}
	baseURL := appPath("/admin/tenants/" + tenantID.String() + "/users/" + userID.String())
	return render(c, profileFieldForm(baseURL, defn, fieldValue))
}

func handleAdminUserProfileFieldEditForm(a App, c *gin.Context) error {
	tenantID, err := rest.UUIDParam(c, "tenantID")
	if err != nil {
		return apperr.NewBadRequestE(err)
	}
	userID, err := rest.UUIDParam(c, "userID")
	if err != nil {
		return apperr.NewBadRequestE(err)
	}
	ten, err := db.GetTenantByID(c.Request.Context(), a.DDB(), tenantID)
	if err != nil {
		return apperr.NewNotFoundE(err)
	}
	fieldName := c.Query("fieldName")
	if fieldName == "" {
		return apperr.NewBadRequest("missing field name")
	}
	defn := ten.GetUserProfileDefinition().GetFieldDefinition(c.Query("fieldName"))
	if defn == nil {
		return apperr.NewBadRequest("invalid field name: " + fieldName)
	}
	targetUser, err := db.GetUserInTenantByID(c.Request.Context(), a.DDB(), tenantID, userID)
	if err != nil {
		zap.L().Error("error fetching user",
			zap.Error(err),
			zap.String("tenantID", tenantID.String()),
			zap.String("userID", userID.String()))
		return apperr.NewUnauthorized("not authorized")
	}
	baseURL := appPath("/admin/tenants/" + tenantID.String() + "/users/" + userID.String())
	return render(c, profileEditFieldForm(baseURL, defn, targetUser.ProfileField(fieldName)))
}

func handlerAdminUserProfileFieldForm(a App, c *gin.Context) error {
	tenantID, err := rest.UUIDParam(c, "tenantID")
	if err != nil {
		return apperr.NewBadRequestE(err)
	}
	userID, err := rest.UUIDParam(c, "userID")
	if err != nil {
		return apperr.NewBadRequestE(err)
	}
	ten, err := db.GetTenantByID(c.Request.Context(), a.DDB(), tenantID)
	if err != nil {
		return apperr.NewNotFoundE(err)
	}
	fieldName := c.Query("fieldName")
	if fieldName == "" {
		return apperr.NewBadRequest("missing field name")
	}
	defn := ten.GetUserProfileDefinition().GetFieldDefinition(c.Query("fieldName"))
	if defn == nil {
		return apperr.NewBadRequest("invalid field name: " + fieldName)
	}
	targetUser, err := db.GetUserInTenantByID(c.Request.Context(), a.DDB(), tenantID, userID)
	if err != nil {
		zap.L().Error("error fetching user",
			zap.Error(err),
			zap.String("tenantID", tenantID.String()),
			zap.String("userID", userID.String()))
		return apperr.NewUnauthorized("not authorized")
	}
	baseURL := appPath("/admin/tenants/" + tenantID.String() + "/users/" + userID.String())
	return render(c, profileFieldForm(baseURL, defn, targetUser.ProfileField(fieldName)))
}

func handleAdminTenantProfileFieldSubmit(a App, c *gin.Context) error {
	tenantID, err := rest.UUIDParam(c, "tenantID")
	if err != nil {
		return err
	}
	fieldName := c.Query("fieldName")
	fieldValue := c.PostForm("fieldValue")
	req := rest.UpdateProfileFieldsRequest{}
	req.ProfileFields = append(req.ProfileFields, rest.ProfileField{
		Name:  fieldName,
		Value: fieldValue,
	})
	err = rest.HandleUpdateTenantProfileFields(a, c, tenantID, req)
	if err != nil {
		return apperr.NewInternalE(err)
	}
	targetTenant, err := db.GetTenantByID(c.Request.Context(), a.DDB(), tenantID)
	if err != nil {
		return apperr.NewInternalE(err)
	}
	defn := targetTenant.GetTenantProfileDefinition().GetFieldDefinition(fieldName)
	if defn == nil {
		return apperr.NewBadRequest("invalid field name: " + fieldName)
	}
	baseURL := appPath("/admin/tenants/" + tenantID.String())
	return render(c, profileFieldForm(baseURL, defn, fieldValue))
}

func handleAdminTenantProfileFieldEditForm(a App, c *gin.Context) error {
	tenantID, err := rest.UUIDParam(c, "tenantID")
	if err != nil {
		return apperr.NewBadRequestE(err)
	}
	targetTenant, err := db.GetTenantByID(c.Request.Context(), a.DDB(), tenantID)
	if err != nil {
		return apperr.NewNotFoundE(err)
	}
	fieldName := c.Query("fieldName")
	if fieldName == "" {
		return apperr.NewBadRequest("missing field name")
	}
	defn := targetTenant.GetTenantProfileDefinition().GetFieldDefinition(c.Query("fieldName"))
	if defn == nil {
		return apperr.NewBadRequest("invalid field name: " + fieldName)
	}
	baseURL := appPath("/admin/tenants/" + tenantID.String())
	return render(c, profileEditFieldForm(baseURL, defn, targetTenant.ProfileField(fieldName)))
}

func handlerAdminTenantProfileFieldForm(a App, c *gin.Context) error {
	tenantID, err := rest.UUIDParam(c, "tenantID")
	if err != nil {
		return apperr.NewBadRequestE(err)
	}
	targetTenant, err := db.GetTenantByID(c.Request.Context(), a.DDB(), tenantID)
	if err != nil {
		return apperr.NewNotFoundE(err)
	}
	fieldName := c.Query("fieldName")
	if fieldName == "" {
		return apperr.NewBadRequest("missing field name")
	}
	defn := targetTenant.GetTenantProfileDefinition().GetFieldDefinition(c.Query("fieldName"))
	if defn == nil {
		return apperr.NewBadRequest("invalid field name: " + fieldName)
	}
	baseURL := appPath("/admin/tenants/" + tenantID.String())
	return render(c, profileFieldForm(baseURL, defn, targetTenant.ProfileField(fieldName)))
}
