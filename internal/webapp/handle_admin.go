package webapp

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"slices"
	"sort"

	"github.com/apresdev/landlord/internal/timers"

	"github.com/apresdev/landlord/internal/tinytools"

	"github.com/apresdev/landlord/internal/perms"

	"github.com/apresdev/landlord/internal/apperr"
	"github.com/apresdev/landlord/internal/db"
	"github.com/apresdev/landlord/internal/multi"
	"github.com/apresdev/landlord/internal/rest"
	"github.com/google/uuid"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func handleAdminDashboard(a App, c *gin.Context) error {
	_ = a
	redirectTo(c, "/admin/tenants")
	return nil
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
	ev := NewErrorPageView(a, errMsg)
	err := executeErrorTemplate(c, "error", ev)
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
	sv := NewAuthenticatedSubjectView(c.Request.Context(), a, sub).WithTenants(tens)
	return executeAdminTemplate(c, "tenants", sv)
}

func handleAdminCreateTenantButton(a App, c *gin.Context) error {
	_ = a
	return executeFragmentTemplate(c, "createTenantButton", NewFragmentView())
}

func handleAdminCreateTenantForm(a App, c *gin.Context) error {
	_ = a
	return executeFragmentTemplate(c, "createTenantForm", NewFragmentView())
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
	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		return err
	}
	sv := NewAuthenticatedSubjectView(c.Request.Context(), a, sub).
		WithTenants(tens).
		WithErrMsg(errMsg)
	return executeAdminTemplate(c, "createTenantTableAndForm", sv)
}

func handleAdminActivateTenantSubmit(a App, c *gin.Context) error {
	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		return err
	}
	i := c.Param("tenantID")
	if i == "" {
		return fmt.Errorf("no id provided")
	}
	ten, err := rest.HandleUpdateTenantState(a, c, i, string(multi.TenantStateActive))
	if err != nil {
		return err
	}
	sv := NewAuthenticatedSubjectView(c.Request.Context(), a, sub).
		WithSelectedTen(ten)
	return executeTenantTemplate(c, "tenantFragment", sv.Tenant())
}

func handleAdminDeactivateTenantSubmit(a App, c *gin.Context) error {
	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		return err
	}
	i := c.Param("tenantID")
	if i == "" {
		return fmt.Errorf("no id provided")
	}
	ten, err := rest.HandleUpdateTenantState(a, c, i, string(multi.TenantStateInactive))
	if err != nil {
		return err
	}
	sv := NewAuthenticatedSubjectView(c.Request.Context(), a, sub).
		WithSelectedTen(ten)
	return executeTenantTemplate(c, "tenantFragment", sv.Tenant())
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
	sv := NewAuthenticatedSubjectView(c.Request.Context(), a, sub).
		WithSelectedTen(ten).
		WithTenantTab(startingTab)
	return executeAdminTemplate(c, "tenant", sv)
}

func usersForTenant(ctx context.Context, a App, ten *multi.Tenant) ([]*multi.User, error) {
	users, err := db.GetUsersForTenant(ctx, a.DDB(), ten.ID)
	if err != nil {
		return nil, apperr.NewInternalE(
			fmt.Errorf("error getting users for tenant: %w", err))
	}
	sort.Slice(users, func(i, j int) bool {
		return users[i].Username < users[j].Username
	})
	return users, nil
}

func handleAdminTenantTab(a App, c *gin.Context) error {
	timers.Checkpoint(c, "handleAdminTenantTab")
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

	tab := newTenantTab(c.Param("tab"))
	sv := NewAuthenticatedSubjectView(c.Request.Context(), a, sub).
		WithSelectedTen(ten).
		WithTenantTab(tab)
	timers.Checkpoint(c, "handleAdminTenantTab.switch")
	defer timers.Checkpoint(c, "handleAdminTenantTab.done")
	switch tab {
	case tenantSummaryTab:
		return executeAdminTemplate(c, tab.TabTemplateName(), sv)
	case tenantProfileTab:
		return executeAdminTemplate(c, tab.TabTemplateName(), sv)
	case tenantUsersTab:
		users, err := usersForTenant(c.Request.Context(), a, ten)
		if err != nil {
			return err
		}
		sv = sv.WithUsers(users)
		timers.Checkpoint(c, "handleAdminTenantTab.withUsers")
		return executeAdminTemplate(c, tab.TabTemplateName(), sv)
	case tenantInvitesTab:
		users, err := usersForTenant(c.Request.Context(), a, ten)
		if err != nil {
			return err
		}
		invites := make([]*multi.User, 0, len(users))
		for _, user := range users {
			if user.Status == multi.UserStatusInvited {
				invites = append(invites, user)
			}
		}
		sv = sv.WithUsers(invites)
		return executeAdminTemplate(c, tab.TabTemplateName(), sv)
	case tenantRolesTab:
		users, err := usersForTenant(c.Request.Context(), a, ten)
		if err != nil {
			return err
		}
		sv = sv.WithUsers(users)
		return executeAdminTemplate(c, tab.TabTemplateName(), sv)
	default:
		zap.L().Info("unknown tenant tab", zap.String("tab", string(tab)))
		return executeAdminTemplate(c, tenantSummaryTab.TabTemplateName(), sv)
	}
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
	return executeTenantTemplate(
		c, "tenantDescriptionForm", NewTenantView(c.Request.Context(), a, ten))
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
	return executeTenantTemplate(
		c, "tenantDescriptionEditForm", NewTenantView(c.Request.Context(), a, ten))
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
	return executeTenantTemplate(
		c, "tenantDescriptionForm", NewTenantView(c.Request.Context(), a, ten))
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
	return executeTenantTemplate(
		c, "inviteUserForm", NewTenantView(c.Request.Context(), a, ten))
}

func handleAdminInviteUserButton(a App, c *gin.Context) error {
	tenantID := c.Param("tenantID")
	th, err := multi.ParseTenantHandle(tenantID)
	if err != nil {
		return apperr.NewBadRequestE(err)
	}
	ten, err := db.ResolveTenant(c.Request.Context(), a.DDB(), th)
	if err != nil {
		return apperr.NewNotFoundE(err)
	}
	return executeTenantTemplate(
		c, "inviteUserButton", NewTenantView(c.Request.Context(), a, ten))
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
	remembrances := make([]remembrance, 0)
	for _, profileFieldDef := range ten.GetUserProfileDefinition().Fields {
		v := c.PostForm(profileFieldDef.Name)
		if v != "" {
			remembrances = append(remembrances, remembrance{
				def: profileFieldDef, formValue: v})
		}
	}
	if len(remembrances) > 0 {
		var updateProfileFieldsReq rest.UpdateProfileFieldsRequest
		for _, r := range remembrances {
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
	sv := NewAuthenticatedSubjectView(c.Request.Context(), a, sub).
		WithSelectedTen(ten).
		WithUsers(users)
	return executeAdminTemplate(c, "inviteUserTableAndForm", sv)
}

func handleAdminAddRoleSubmit(a App, c *gin.Context) error {
	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		return err
	}
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
		sv := NewAuthenticatedSubjectView(c.Request.Context(), a, sub).
			WithSelectedTen(ten).
			WithUsers(users).
			WithErrMsg("Role already exists: " + role)
		return executeAdminTemplate(c, "addRoleTableAndForm", sv)
	} else {
		var req rest.TenantRolesRequest
		req.Roles = append(ten.Roles, role)
		req.SubTenantRoles = ten.SubTenantRoles
		updatedTenant, err := rest.HandleUpdateRolesForTenant(a, c, req, tenantID)
		if err != nil {
			return apperr.NewInternalE(err)
		}
		users, err := db.GetUsersForTenant(c.Request.Context(), a.DDB(), ten.ID)
		if err != nil {
			return apperr.NewInternalE(err)
		}
		zap.L().Info("Adding role to tenant", zap.String("role", role))

		sv := NewAuthenticatedSubjectView(c.Request.Context(), a, sub).
			WithSelectedTen(updatedTenant).
			WithUsers(users)
		return executeAdminTemplate(c, "addRoleTableAndForm", sv)
	}
}

func handleAdminDeleteRoleSubmit(a App, c *gin.Context) error {
	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		return err
	}
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
	updatedTenant, err := rest.HandleUpdateRolesForTenant(a, c, req, tenantID)
	if err != nil {
		return apperr.NewInternalE(err)
	}
	users, err := db.GetUsersForTenant(c.Request.Context(), a.DDB(), ten.ID)
	if err != nil {
		return apperr.NewInternalE(err)
	}
	zap.L().Info("Removed role from tenant", zap.String("role", role))
	sv := NewAuthenticatedSubjectView(c.Request.Context(), a, sub).
		WithSelectedTen(updatedTenant).
		WithUsers(users)
	return executeAdminTemplate(c, "addRoleTableAndForm", sv)
}

func handleAdminAddRoleForm(a App, c *gin.Context) error {
	_ = a
	return executeStringTemplate(c, "tenantAddRoleForm", c.Param("tenantID"))
}

func handleAdminAddRoleButton(a App, c *gin.Context) error {
	_ = a
	return executeStringTemplate(c, "tenantAddRoleButton", c.Param("tenantID"))
}

func handleAdminAddSubRoleSubmit(a App, c *gin.Context) error {
	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		return err
	}
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
		sv := NewAuthenticatedSubjectView(c.Request.Context(), a, sub).
			WithSelectedTen(ten).
			WithErrMsg("Role already exists: " + role)
		return executeAdminTemplate(c, "addSubRoleTableAndForm", sv)
	} else {
		var req rest.TenantRolesRequest
		req.Roles = ten.Roles
		req.SubTenantRoles = append(ten.SubTenantRoles, role)
		updatedTenant, err := rest.HandleUpdateRolesForTenant(a, c, req, tenantID)
		if err != nil {
			return apperr.NewInternalE(err)
		}
		zap.L().Info("Adding sub-role to tenant", zap.String("sub-role", role))
		sv := NewAuthenticatedSubjectView(c.Request.Context(), a, sub).
			WithSelectedTen(updatedTenant)
		return executeAdminTemplate(c, "addSubRoleTableAndForm", sv)
	}
}

func handleAdminDeleteSubRoleSubmit(a App, c *gin.Context) error {
	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		return err
	}
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
	updatedTenant, err := rest.HandleUpdateRolesForTenant(a, c, req, tenantID)
	if err != nil {
		return apperr.NewInternalE(err)
	}
	zap.L().Info("Removed sub-role from tenant", zap.String("sub-role", subRole))
	sv := NewAuthenticatedSubjectView(c.Request.Context(), a, sub).
		WithSelectedTen(updatedTenant)
	return executeAdminTemplate(c, "addSubRoleTableAndForm", sv)
}

func handleAdminAddSubRoleForm(a App, c *gin.Context) error {
	_ = a
	return executeStringTemplate(c, "addSubRoleForm", c.Param("tenantID"))
}

func handleAdminAddSubRoleButton(a App, c *gin.Context) error {
	_ = a
	return executeStringTemplate(c, "addSubRoleButton", c.Param("tenantID"))
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
	return executeUserTemplate(
		c, "userFragment", NewUserView(c.Request.Context(), a, sub, ten, user))
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
	return executeUserTemplate(
		c, "userFragment", NewUserView(c.Request.Context(), a, sub, ten, user))
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
	return executeUserTemplate(
		c, "userFragment", NewUserView(c.Request.Context(), a, sub, ten, user))
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
	return executeUserTemplate(
		c, "userFragment", NewUserView(c.Request.Context(), a, sub, ten, user))
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
	user, err := db.GetUserInTenantByID(c.Request.Context(), a.DDB(), ten.ID, userUUID)
	if err != nil {
		return apperr.NewNotFoundE(
			fmt.Errorf("error getting user: %w", err))
	}
	sv := NewAuthenticatedSubjectView(c.Request.Context(), a, sub).
		WithSelectedTen(ten).
		WithSelectedUser(user).
		WithUserTab(userSummaryTab)
	return executeAdminTemplate(c, "user", sv)
}

func handleAdminProfile(a App, c *gin.Context) error {
	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		clearLandlordCookies(c, a.Settings().ConsoleURL())
		redirectToLogin(c)
		return nil
	}
	sv := NewAuthenticatedSubjectView(c.Request.Context(), a, sub)
	return executeAdminTemplate(c, "profile", sv)
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
		sv := NewAuthenticatedSubjectView(c.Request.Context(), a, sub).
			WithSelectedTen(subjectTenant).
			WithSelectedUser(subjectUser).
			WithProfileTab(tab)
		return executeAdminTemplate(c, "profileSummaryTab", sv)
	case profileTabDetails:
		subjectTenant, err := sub.ResolveTenant(c.Request.Context(), a)
		if err != nil {
			return err
		}
		subjectUser, err := sub.ResolveUser(c.Request.Context(), a)
		if err != nil {
			return err
		}
		sv := NewAuthenticatedSubjectView(c.Request.Context(), a, sub).
			WithSelectedTen(subjectTenant).
			WithSelectedUser(subjectUser).
			WithProfileTab(tab)
		return executeAdminTemplate(c, "profileDetailsTab", sv)
	case profileTabAccessToken:
		sv := NewAuthenticatedSubjectView(c.Request.Context(), a, sub).
			WithProfileTab(tab)
		return executeAdminTemplate(c, "profileTokenTab", sv)
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
	def := ten.GetUserProfileDefinition().GetFieldDefinition(fieldName)
	if def == nil {
		return apperr.NewBadRequest("invalid field name: " + fieldName)
	}
	targetUser, err := db.GetUserInTenantByID(c.Request.Context(), a.DDB(), tenantID, userID)
	if err != nil {
		return apperr.NewInternalE(err)
	}
	baseURL := appPath("/admin/tenants/" + tenantID.String() + "/users/" + userID.String())
	pfv := NewProfileFieldView(baseURL, *def, targetUser.ProfileField(fieldName))
	return executeProfileFieldTemplate(c, "profileFieldForm", pfv)
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
	def := ten.GetUserProfileDefinition().GetFieldDefinition(c.Query("fieldName"))
	if def == nil {
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
	pfv := NewProfileFieldView(baseURL, *def, targetUser.ProfileField(fieldName))
	return executeProfileFieldTemplate(c, "profileEditFieldForm", pfv)
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
	def := ten.GetUserProfileDefinition().GetFieldDefinition(c.Query("fieldName"))
	if def == nil {
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
	pfv := NewProfileFieldView(baseURL, *def, targetUser.ProfileField(fieldName))
	return executeProfileFieldTemplate(c, "profileFieldForm", pfv)
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
	def := targetTenant.GetTenantProfileDefinition().GetFieldDefinition(fieldName)
	if def == nil {
		return apperr.NewBadRequest("invalid field name: " + fieldName)
	}
	baseURL := appPath("/admin/tenants/" + tenantID.String())
	pfv := NewProfileFieldView(baseURL, *def, targetTenant.ProfileField(fieldName))
	return executeProfileFieldTemplate(c, "profileFieldForm", pfv)
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
	def := targetTenant.GetTenantProfileDefinition().GetFieldDefinition(c.Query("fieldName"))
	if def == nil {
		return apperr.NewBadRequest("invalid field name: " + fieldName)
	}
	baseURL := appPath("/admin/tenants/" + tenantID.String())
	pfv := NewProfileFieldView(baseURL, *def, targetTenant.ProfileField(fieldName))
	return executeProfileFieldTemplate(c, "profileEditFieldForm", pfv)
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
	def := targetTenant.GetTenantProfileDefinition().GetFieldDefinition(c.Query("fieldName"))
	if def == nil {
		return apperr.NewBadRequest("invalid field name: " + fieldName)
	}
	baseURL := appPath("/admin/tenants/" + tenantID.String())
	pfv := NewProfileFieldView(baseURL, *def, targetTenant.ProfileField(fieldName))
	return executeProfileFieldTemplate(c, "profileFieldForm", pfv)
}

func handleAdminUserTab(a App, c *gin.Context) error {
	timers.Checkpoint(c, "handleAdminUserTab")
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
	user, err := db.GetUserInTenantByID(c.Request.Context(), a.DDB(), ten.ID, userUUID)
	if err != nil {
		return apperr.NewNotFoundE(
			fmt.Errorf("error getting user: %w", err))
	}
	tab := newUserTab(c.Param("tab"))
	sv := NewAuthenticatedSubjectView(c.Request.Context(), a, sub).
		WithSelectedTen(ten).
		WithSelectedUser(user).
		WithUserTab(tab)

	timers.Checkpoint(c, "handleAdminUserTab.switch")
	defer timers.Checkpoint(c, "handleAdminUserTab.done")
	switch tab {
	case userSummaryTab:
		return executeAdminTemplate(c, tab.TabTemplateName(), sv)
	case userProfileTab:
		return executeAdminTemplate(c, tab.TabTemplateName(), sv)
	default:
		zap.L().Info("unknown tenant tab", zap.String("tab", string(tab)))
		return executeAdminTemplate(c, userSummaryTab.TabTemplateName(), sv)
	}
}

func handleAdminUserRoleSubmit(a App, c *gin.Context) error {
	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		clearLandlordCookies(c, a.Settings().ConsoleURL())
		redirectToLogin(c)
		return nil
	}
	tenantID, err := rest.UUIDParam(c, "tenantID")
	if err != nil {
		return err
	}
	userID, err := rest.UUIDParam(c, "userID")
	if err != nil {
		return err
	}
	role := c.PostForm("role")
	targetTenant, targetUser, err := rest.HandleUpdateUserRole(a, c, sub, role, tenantID, userID)
	if err != nil {
		return apperr.NewInternalE(err)
	}
	uv := NewUserView(c.Request.Context(), a, sub, targetTenant, targetUser)
	return executeUserTemplate(c, "userRoleForm", uv)
}

func handleAdminUserRoleEditForm(a App, c *gin.Context) error {
	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		clearLandlordCookies(c, a.Settings().ConsoleURL())
		redirectToLogin(c)
		return nil
	}
	tenantID, err := rest.UUIDParam(c, "tenantID")
	if err != nil {
		return apperr.NewBadRequestE(err)
	}
	userID, err := rest.UUIDParam(c, "userID")
	if err != nil {
		return apperr.NewBadRequestE(err)
	}
	targetTenant, err := db.GetTenantByID(c.Request.Context(), a.DDB(), tenantID)
	if err != nil {
		return apperr.NewNotFoundE(err)
	}
	targetUser, err := db.GetUserInTenantByID(c.Request.Context(), a.DDB(), tenantID, userID)
	if err != nil {
		zap.L().Error("error fetching user",
			zap.Error(err),
			zap.String("tenantID", tenantID.String()),
			zap.String("userID", userID.String()))
		return apperr.NewUnauthorized("not authorized")
	}
	uv := NewUserView(c.Request.Context(), a, sub, targetTenant, targetUser)
	return executeUserTemplate(c, "userRoleEditForm", uv)
}

func handlerAdminUserRoleForm(a App, c *gin.Context) error {
	tenantID, err := rest.UUIDParam(c, "tenantID")
	if err != nil {
		return apperr.NewBadRequestE(err)
	}
	userID, err := rest.UUIDParam(c, "userID")
	if err != nil {
		return apperr.NewBadRequestE(err)
	}
	targetTenant, err := db.GetTenantByID(c.Request.Context(), a.DDB(), tenantID)
	if err != nil {
		return apperr.NewNotFoundE(err)
	}
	targetUser, err := db.GetUserInTenantByID(c.Request.Context(), a.DDB(), tenantID, userID)
	if err != nil {
		zap.L().Error("error fetching user",
			zap.Error(err),
			zap.String("tenantID", tenantID.String()),
			zap.String("userID", userID.String()))
		return apperr.NewUnauthorized("not authorized")
	}
	uv := NewUserView(c.Request.Context(), a, nil, targetTenant, targetUser)
	return executeUserTemplate(c, "userRoleForm", uv)
}
