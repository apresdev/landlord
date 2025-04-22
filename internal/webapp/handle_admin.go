package webapp

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/apresdev/landlord/internal/timers"

	"github.com/apresdev/landlord/internal/tinytools"

	"github.com/apresdev/landlord/internal/perms"

	"github.com/apresdev/landlord/internal/apperr"
	"github.com/apresdev/landlord/internal/db"
	"github.com/apresdev/landlord/internal/multi"
	"github.com/apresdev/landlord/internal/rest"
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

func handleAdminAbout(a App, c *gin.Context, sub *perms.Subject) error {
	sv := NewAuthenticatedSubjectView(c.Request.Context(), a, sub)
	return executeAdminTemplate(c, "about", sv)
}

func handleAdminClearImpersonation(a App, c *gin.Context, sub *perms.Subject) error {
	err := rest.HandleClearImpersonation(a, c, sub)
	if err != nil {
		return err
	}
	clearLandlordCookies(c, a.Settings().ConsoleURL())
	c.Header("HX-Redirect", appPath("/"))
	c.Status(http.StatusNoContent)
	return nil
}

func handleAdminTenants(a App, c *gin.Context, sub *perms.Subject) error {
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

func handleAdminCreateTenantSubmit(a App, c *gin.Context, sub *perms.Subject) error {
	var req rest.CreateTenantRequest
	req.Name = c.PostForm("name")
	req.Description = c.PostForm("description")
	_, err := rest.HandleCreateTenant(a, c, req)
	errMsg := ""
	if err != nil {
		errMsg = fmt.Sprintf("Could not create tenant: " + err.Error())
	}
	tens, err := rest.HandleGetTenants(a, c)
	if err != nil {
		return err
	}
	sv := NewAuthenticatedSubjectView(c.Request.Context(), a, sub).
		WithTenants(tens).
		WithErrMsg(errMsg)
	return executeAdminTemplate(c, "createTenantTableAndForm", sv)
}

func handleAdminActivateTenantSubmit(
	a App, c *gin.Context, sub *perms.Subject, targetTenant *multi.Tenant) error {

	resultTenant, err := rest.HandleUpdateTenantState(
		a, c, sub, targetTenant, string(multi.TenantStateActive))
	if err != nil {
		return err
	}
	sv := NewAuthenticatedSubjectView(c.Request.Context(), a, sub).
		WithSelectedTen(resultTenant)
	return executeTenantTemplate(c, "tenantFragment", sv.Tenant())
}

func handleAdminDeactivateTenantSubmit(
	a App, c *gin.Context, sub *perms.Subject, targetTenant *multi.Tenant) error {

	resultTenant, err := rest.HandleUpdateTenantState(
		a, c, sub, targetTenant, string(multi.TenantStateInactive))
	if err != nil {
		return err
	}
	sv := NewAuthenticatedSubjectView(c.Request.Context(), a, sub).
		WithSelectedTen(resultTenant)
	return executeTenantTemplate(c, "tenantFragment", sv.Tenant())
}

func handleAdminTenant(
	a App, c *gin.Context, sub *perms.Subject, targetTenant *multi.Tenant) error {

	startingTab := newTenantTab(c.Query("tab"))
	sv := NewAuthenticatedSubjectView(c.Request.Context(), a, sub).
		WithSelectedTen(targetTenant).
		WithTenantTab(startingTab)
	return executeAdminTemplate(c, "tenant", sv)
}

func usersForTenant(ctx context.Context, a App, ten *multi.Tenant) ([]*multi.User, error) {
	users, err := db.GetUsersForTenant(ctx, a.DBCtx(), ten.ID)
	if err != nil {
		return nil, apperr.NewInternalE(
			fmt.Errorf("error getting users for tenant: %w", err))
	}
	sort.Slice(users, func(i, j int) bool {
		return users[i].Username < users[j].Username
	})
	return users, nil
}

func handleAdminTenantTab(
	a App, c *gin.Context, sub *perms.Subject, targetTenant *multi.Tenant) error {

	tab := newTenantTab(c.Param("tab"))
	sv := NewAuthenticatedSubjectView(c.Request.Context(), a, sub).
		WithSelectedTen(targetTenant).
		WithTenantTab(tab)
	timers.Checkpoint(c, "handleAdminTenantTab.switch")
	defer timers.Checkpoint(c, "handleAdminTenantTab.done")
	switch tab {
	case tenantSummaryTab:
		return executeAdminTemplate(c, tab.TabTemplateName(), sv)
	case tenantProfileTab:
		return executeAdminTemplate(c, tab.TabTemplateName(), sv)
	case tenantUsersTab:
		users, err := usersForTenant(c.Request.Context(), a, targetTenant)
		if err != nil {
			return err
		}
		sv = sv.WithUsers(users)
		timers.Checkpoint(c, "handleAdminTenantTab.withUsers")
		return executeAdminTemplate(c, tab.TabTemplateName(), sv)
	case tenantInvitesTab:
		users, err := usersForTenant(c.Request.Context(), a, targetTenant)
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
		users, err := usersForTenant(c.Request.Context(), a, targetTenant)
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

func handleAdminDescriptionSubmit(
	a App, c *gin.Context, sub *perms.Subject, targetTenant *multi.Tenant) error {

	ten, err := rest.HandleUpdateTenantDescription(
		a, c, sub, targetTenant, c.PostForm("description"))
	if err != nil {
		return apperr.NewInternalE(err)
	}
	return executeTenantTemplate(
		c, "tenantDescriptionForm", NewTenantView(c.Request.Context(), a, sub, ten))
}

func handleAdminDescriptionEditForm(
	a App, c *gin.Context, sub *perms.Subject, targetTenant *multi.Tenant) error {

	_ = sub
	return executeTenantTemplate(
		c, "tenantDescriptionEditForm",
		NewTenantView(c.Request.Context(), a, sub, targetTenant))
}

func handleAdminDescriptionForm(
	a App, c *gin.Context, sub *perms.Subject, targetTenant *multi.Tenant) error {

	_ = sub
	return executeTenantTemplate(
		c, "tenantDescriptionForm",
		NewTenantView(c.Request.Context(), a, sub, targetTenant))
}

func handleAdminInviteUserForm(
	a App, c *gin.Context, sub *perms.Subject, targetTenant *multi.Tenant) error {

	_ = sub
	return executeTenantTemplate(
		c, "inviteUserForm",
		NewTenantView(c.Request.Context(), a, sub, targetTenant))
}

func handleInviteButtons(a App, c *gin.Context) error {
	_ = a
	tenantID, err := rest.UUIDParam(c, "tenantID")
	if err != nil {
		return apperr.NewBadRequestE(err)
	}
	return executeStringTemplate(c, "inviteButtons", tenantID.String())
}

func inviteUserFormError(
	a App, c *gin.Context, sub *perms.Subject, tenant *multi.Tenant, errMsg string) error {

	c.Header("HX-Retarget", "#invite-form")
	tv := NewTenantView(c.Request.Context(), a, sub, tenant).
		WithErrMsg(errMsg)
	return executeTenantTemplate(c, "inviteUserForm", tv)
}

func handleAdminInviteUserSubmit(
	a App, c *gin.Context, sub *perms.Subject, targetTenant *multi.Tenant) error {

	var req rest.InviteRequest
	req.Username = c.PostForm("username")
	req.Role = c.PostForm("role")
	req.MfaPolicy = rest.MfaPolicy(c.PostForm("mfaPolicy"))

	if err := rest.HandleInviteToTenant(a, c, sub, targetTenant, req); err != nil {
		return inviteUserFormError(a, c, sub, targetTenant, err.Error())
	}

	type remembrance struct {
		def       multi.ProfileFieldDefinition
		formValue string
	}
	remembrances := make([]remembrance, 0)
	for _, profileFieldDef := range targetTenant.GetUserProfileDefinition().Fields {
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
					Name:  r.def.UIName,
					Value: r.formValue})
		}
		user, err := db.GetUserByName(c.Request.Context(), a.DBCtx(), req.Username)
		if err != nil {
			return apperr.NewNotFoundE(err)
		}
		err = rest.HandleUpdateUserProfileFields(
			a, c, sub, targetTenant, user, updateProfileFieldsReq)
		if err != nil {
			return inviteUserFormError(a, c, sub, targetTenant,
				"Could not update profile fields during invite: "+err.Error())
		}
	}

	users, err := db.GetUsersForTenant(c.Request.Context(), a.DBCtx(), targetTenant.ID)
	if err != nil {
		return apperr.NewInternalE(err)
	}
	sv := NewAuthenticatedSubjectView(c.Request.Context(), a, sub).
		WithSelectedTen(targetTenant).
		WithUsers(users)
	return executeAdminTemplate(c, "inviteUserTableAndForm", sv)
}

func handleAdminAddRoleSubmit(
	a App, c *gin.Context, sub *perms.Subject, targetTenant *multi.Tenant) error {

	role := c.PostForm("role")
	if slices.Contains(targetTenant.Roles, role) {
		users, err := db.GetUsersForTenant(c.Request.Context(), a.DBCtx(), targetTenant.ID)
		if err != nil {
			return apperr.NewInternalE(err)
		}
		sv := NewAuthenticatedSubjectView(c.Request.Context(), a, sub).
			WithSelectedTen(targetTenant).
			WithUsers(users).
			WithErrMsg("Role already exists: " + role)
		return executeAdminTemplate(c, "addRoleTableAndForm", sv)
	} else {
		var req rest.TenantRolesRequest
		req.Roles = append(targetTenant.Roles, role)
		req.SubTenantRoles = targetTenant.SubTenantRoles
		updatedTenant, err := rest.HandleUpdateRolesForTenant(a, c, sub, targetTenant, req)
		if err != nil {
			return apperr.NewInternalE(err)
		}
		users, err := db.GetUsersForTenant(c.Request.Context(), a.DBCtx(), targetTenant.ID)
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

func handleAdminImpersonateRoleSubmit(
	a App, c *gin.Context, sub *perms.Subject, targetTenant *multi.Tenant) error {

	role := c.Param("role")
	refreshToken, _ := c.Cookie(LandlordRefreshTokenCookie)
	_, err := rest.HandleImpersonateRole(a, c, sub, targetTenant, role, refreshToken)
	if err != nil {
		return err
	}
	clearLandlordCookies(c, a.Settings().ConsoleURL())
	c.Header("HX-Redirect", appPath("/"))
	c.Status(http.StatusNoContent)
	return nil
}

func handleAdminDeleteRoleSubmit(
	a App, c *gin.Context, sub *perms.Subject, targetTenant *multi.Tenant) error {

	role := c.Param("role")
	if !slices.Contains(targetTenant.Roles, role) {
		return apperr.NewBadRequestE(fmt.Errorf("role does not exist: %s", role))
	}
	var req rest.TenantRolesRequest
	req.Roles = tinytools.RemoveFromSlice(targetTenant.Roles, role)
	req.SubTenantRoles = targetTenant.SubTenantRoles
	updatedTenant, err := rest.HandleUpdateRolesForTenant(a, c, sub, targetTenant, req)
	if err != nil {
		return apperr.NewInternalE(err)
	}
	users, err := db.GetUsersForTenant(c.Request.Context(), a.DBCtx(), targetTenant.ID)
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
	return executeUUIDTemplate(c, "tenantAddRoleForm", "tenantID")
}

func handleAdminAddRoleButton(a App, c *gin.Context) error {
	_ = a
	return executeUUIDTemplate(c, "tenantAddRoleButton", "tenantID")
}

func handleAdminAddSubRoleSubmit(
	a App, c *gin.Context, sub *perms.Subject, targetTenant *multi.Tenant) error {

	role := c.PostForm("subrole")
	if slices.Contains(targetTenant.SubTenantRoles, role) {
		sv := NewAuthenticatedSubjectView(c.Request.Context(), a, sub).
			WithSelectedTen(targetTenant).
			WithErrMsg("Role already exists: " + role)
		return executeAdminTemplate(c, "addSubRoleTableAndForm", sv)
	} else {
		var req rest.TenantRolesRequest
		req.Roles = targetTenant.Roles
		req.SubTenantRoles = append(targetTenant.SubTenantRoles, role)
		updatedTenant, err := rest.HandleUpdateRolesForTenant(a, c, sub, targetTenant, req)
		if err != nil {
			return apperr.NewInternalE(err)
		}
		zap.L().Info("Adding sub-role to tenant", zap.String("sub-role", role))
		sv := NewAuthenticatedSubjectView(c.Request.Context(), a, sub).
			WithSelectedTen(updatedTenant)
		return executeAdminTemplate(c, "addSubRoleTableAndForm", sv)
	}
}

func handleAdminDeleteSubRoleSubmit(
	a App, c *gin.Context, sub *perms.Subject, targetTenant *multi.Tenant) error {

	subRole := c.Param("subrole")
	if !slices.Contains(targetTenant.SubTenantRoles, subRole) {
		return apperr.NewBadRequestE(fmt.Errorf("sub-role does not exist: %s", subRole))
	}
	var req rest.TenantRolesRequest
	req.Roles = targetTenant.Roles
	req.SubTenantRoles = tinytools.RemoveFromSlice(targetTenant.SubTenantRoles, subRole)
	updatedTenant, err := rest.HandleUpdateRolesForTenant(a, c, sub, targetTenant, req)
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
	return executeUUIDTemplate(c, "addSubRoleForm", "tenantID")
}

func handleAdminAddSubRoleButton(a App, c *gin.Context) error {
	_ = a
	return executeUUIDTemplate(c, "addSubRoleButton", "tenantID")
}

func handleAdminActivateUserSubmit(
	a App,
	c *gin.Context,
	sub *perms.Subject,
	targetTenant *multi.Tenant,
	targetUser *multi.User) error {

	user, err := rest.HandleUpdateUserStatus(
		a, c, sub, targetTenant, targetUser, string(multi.TenantStateActive))
	if err != nil {
		return err
	}
	ten, err := db.GetTenantByID(c.Request.Context(), a.DBCtx(), user.TenantID)
	if err != nil {
		return apperr.NewInternalE(err)
	}
	return executeUserTemplate(
		c, "userFragment", NewUserView(c.Request.Context(), a, sub, ten, user))
}

func handleAdminDeactivateUserSubmit(
	a App,
	c *gin.Context,
	sub *perms.Subject,
	targetTenant *multi.Tenant,
	targetUser *multi.User) error {

	resultUser, err := rest.HandleUpdateUserStatus(
		a, c, sub, targetTenant, targetUser, string(multi.TenantStateInactive))
	if err != nil {
		return err
	}
	return executeUserTemplate(
		c, "userFragment",
		NewUserView(c.Request.Context(), a, sub, targetTenant, resultUser))
}

func handleAdminCancelInviteSubmit(
	a App,
	c *gin.Context,
	sub *perms.Subject,
	targetTenant *multi.Tenant,
	targetUser *multi.User) error {

	if err := rest.HandleCancelInvitation(a, c, sub, targetTenant, targetUser); err != nil {
		return err
	}
	c.String(http.StatusOK, "<div></div>")
	return nil
}

func handleAdminUpdateMFAPolicy(
	a App,
	c *gin.Context,
	sub *perms.Subject,
	targetTenant *multi.Tenant,
	targetUser *multi.User,
	policy rest.MfaPolicy) error {

	req := rest.UpdateUserMfaRequest{}
	req.MfaPolicy = policy
	resultUser, err := rest.HandleUpdateUserMFA(
		a, c, sub, targetTenant, targetUser, req)
	if err != nil {
		return err
	}
	return executeUserTemplate(
		c, "userFragment",
		NewUserView(c.Request.Context(), a, sub, targetTenant, resultUser))
}

func handleAdminEnableMFASubmit(
	a App,
	c *gin.Context,
	sub *perms.Subject,
	targetTenant *multi.Tenant,
	targetUser *multi.User) error {
	return handleAdminUpdateMFAPolicy(
		a, c, sub, targetTenant, targetUser, rest.MFAPolicyRequired)
}

func handleAdminDisableMFASubmit(
	a App,
	c *gin.Context,
	sub *perms.Subject,
	targetTenant *multi.Tenant,
	targetUser *multi.User) error {

	return handleAdminUpdateMFAPolicy(
		a, c, sub, targetTenant, targetUser, rest.MFAPolicyNone)
}

func handleAdminReInviteUserSubmit(
	a App,
	c *gin.Context,
	sub *perms.Subject,
	targetTenant *multi.Tenant,
	targetUser *multi.User) error {

	var req rest.InviteRequest
	req.Username = targetUser.Username
	req.Role = targetUser.Role
	req.MfaPolicy = rest.MfaPolicy(targetUser.MFAPolicy.String())
	if err := rest.HandleInviteToTenant(a, c, sub, targetTenant, req); err != nil {
		return err
	}
	return executeUserTemplate(
		c, "userFragment",
		NewUserView(c.Request.Context(), a, sub, targetTenant, targetUser))
}

func handleAdminImpersonateUserSubmit(
	a App,
	c *gin.Context,
	sub *perms.Subject,
	targetTenant *multi.Tenant,
	targetUser *multi.User) error {

	refreshToken, _ := c.Cookie(LandlordRefreshTokenCookie)
	_, err := rest.HandleImpersonateUser(a, c, sub, targetTenant, targetUser, refreshToken)
	if err != nil {
		return err
	}
	clearLandlordCookies(c, a.Settings().ConsoleURL())
	c.Header("HX-Redirect", appPath("/"))
	c.Status(http.StatusNoContent)
	return nil
}

func handleAdminUser(
	a App,
	c *gin.Context,
	sub *perms.Subject,
	targetTenant *multi.Tenant,
	targetUser *multi.User) error {

	// TODO: read permissions?

	sv := NewAuthenticatedSubjectView(c.Request.Context(), a, sub).
		WithSelectedTen(targetTenant).
		WithSelectedUser(targetUser).
		WithUserTab(userSummaryTab)
	return executeAdminTemplate(c, "user", sv)
}

func handleAdminProfile(a App, c *gin.Context, sub *perms.Subject) error {
	sv := NewAuthenticatedSubjectView(c.Request.Context(), a, sub)
	return executeAdminTemplate(c, "profile", sv)
}

func handleAdminProfileTab(a App, c *gin.Context, sub *perms.Subject) error {
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

func handleAdminUserProfileFieldSubmit(
	a App,
	c *gin.Context,
	sub *perms.Subject,
	targetTenant *multi.Tenant,
	targetUser *multi.User) error {

	fieldName := c.Query("fieldName")
	fieldValue := c.PostForm("fieldValue")
	req := rest.UpdateProfileFieldsRequest{}
	req.ProfileFields = append(req.ProfileFields, rest.ProfileField{
		Name:  fieldName,
		Value: fieldValue,
	})
	err := rest.HandleUpdateUserProfileFields(
		a, c, sub, targetTenant, targetUser, req)
	if err != nil {
		return apperr.NewInternalE(err)
	}
	def := targetTenant.GetUserProfileDefinition().GetFieldDefinition(fieldName)
	if def == nil {
		return apperr.NewBadRequest("invalid field name: " + fieldName)
	}
	baseURL := appPath(
		"/admin/tenants/" + targetTenant.ID.String() + "/users/" + targetUser.ID.String())
	pfv := NewProfileFieldView(sub, baseURL, *def, targetUser.ProfileField(fieldName))
	return executeProfileFieldTemplate(c, "profileFieldForm", pfv)
}

func handleAdminUserProfileFieldEditForm(
	a App,
	c *gin.Context,
	sub *perms.Subject,
	targetTenant *multi.Tenant,
	targetUser *multi.User) error {

	_ = a
	_ = sub // TODO: read permissions?

	fieldName := c.Query("fieldName")
	if fieldName == "" {
		return apperr.NewBadRequest("missing field name")
	}
	def := targetTenant.GetUserProfileDefinition().GetFieldDefinition(c.Query("fieldName"))
	if def == nil {
		return apperr.NewBadRequest("invalid field name: " + fieldName)
	}
	baseURL := appPath("/admin/tenants/" + targetTenant.ID.String() + "/users/" + targetUser.ID.String())
	pfv := NewProfileFieldView(sub, baseURL, *def, targetUser.ProfileField(fieldName))
	return executeProfileFieldTemplate(c, "profileEditFieldForm", pfv)
}

func handlerAdminUserProfileFieldForm(
	a App,
	c *gin.Context,
	sub *perms.Subject,
	targetTenant *multi.Tenant,
	targetUser *multi.User) error {

	_ = a
	_ = sub // TODO: permissions?

	fieldName := c.Query("fieldName")
	if fieldName == "" {
		return apperr.NewBadRequest("missing field name")
	}
	def := targetTenant.GetUserProfileDefinition().GetFieldDefinition(c.Query("fieldName"))
	if def == nil {
		return apperr.NewBadRequest("invalid field name: " + fieldName)
	}
	baseURL := appPath("/admin/tenants/" + targetTenant.ID.String() + "/users/" + targetUser.ID.String())
	pfv := NewProfileFieldView(sub, baseURL, *def, targetUser.ProfileField(fieldName))
	return executeProfileFieldTemplate(c, "profileFieldForm", pfv)
}

func handleAdminTenantProfileFieldSubmit(
	a App,
	c *gin.Context,
	sub *perms.Subject,
	targetTenant *multi.Tenant) error {

	fieldName := c.Query("fieldName")
	fieldValue := c.PostForm("fieldValue")
	req := rest.UpdateProfileFieldsRequest{}
	req.ProfileFields = append(req.ProfileFields, rest.ProfileField{
		Name:  fieldName,
		Value: fieldValue,
	})
	resultTenant, err := rest.HandleUpdateTenantProfileFields(a, c, sub, targetTenant, req)
	if err != nil {
		return apperr.NewInternalE(err)
	}
	def := targetTenant.GetTenantProfileDefinition().GetFieldDefinition(fieldName)
	if def == nil {
		return apperr.NewBadRequest("invalid field name: " + fieldName)
	}
	baseURL := appPath("/admin/tenants/" + resultTenant.ID.String())
	pfv := NewProfileFieldView(sub, baseURL, *def, resultTenant.ProfileField(fieldName))
	return executeProfileFieldTemplate(c, "profileFieldForm", pfv)
}

func handleAdminTenantProfileFieldEditForm(
	a App,
	c *gin.Context,
	sub *perms.Subject,
	targetTenant *multi.Tenant) error {

	_ = a
	_ = sub // TODO: permissions?

	fieldName := c.Query("fieldName")
	if fieldName == "" {
		return apperr.NewBadRequest("missing field name")
	}
	def := targetTenant.GetTenantProfileDefinition().GetFieldDefinition(c.Query("fieldName"))
	if def == nil {
		return apperr.NewBadRequest("invalid field name: " + fieldName)
	}
	baseURL := appPath("/admin/tenants/" + targetTenant.ID.String())
	pfv := NewProfileFieldView(sub, baseURL, *def, targetTenant.ProfileField(fieldName))
	return executeProfileFieldTemplate(c, "profileEditFieldForm", pfv)
}

func handlerAdminTenantProfileFieldForm(
	a App,
	c *gin.Context,
	sub *perms.Subject,
	targetTenant *multi.Tenant) error {

	_ = a
	_ = sub // TODO: permissions?

	fieldName := c.Query("fieldName")
	if fieldName == "" {
		return apperr.NewBadRequest("missing field name")
	}
	def := targetTenant.GetTenantProfileDefinition().GetFieldDefinition(c.Query("fieldName"))
	if def == nil {
		return apperr.NewBadRequest("invalid field name: " + fieldName)
	}
	baseURL := appPath("/admin/tenants/" + targetTenant.ID.String())
	pfv := NewProfileFieldView(sub, baseURL, *def, targetTenant.ProfileField(fieldName))
	return executeProfileFieldTemplate(c, "profileFieldForm", pfv)
}

func handleAdminUserTab(
	a App,
	c *gin.Context,
	sub *perms.Subject,
	targetTenant *multi.Tenant,
	targetUser *multi.User) error {

	tab := newUserTab(c.Param("tab"))
	sv := NewAuthenticatedSubjectView(c.Request.Context(), a, sub).
		WithSelectedTen(targetTenant).
		WithSelectedUser(targetUser).
		WithUserTab(tab)

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

func handleAdminUserRoleSubmit(
	a App,
	c *gin.Context,
	sub *perms.Subject,
	targetTenant *multi.Tenant,
	targetUser *multi.User) error {

	role := c.PostForm("role")
	resultTenant, resultUser, err := rest.HandleUpdateUserRole(
		a, c, sub, targetTenant, targetUser, role)
	if err != nil {
		return apperr.NewInternalE(err)
	}
	uv := NewUserView(c.Request.Context(), a, sub, resultTenant, resultUser)
	return executeUserTemplate(c, "userRoleForm", uv)
}

func handleAdminUserRoleEditForm(
	a App,
	c *gin.Context,
	sub *perms.Subject,
	targetTenant *multi.Tenant,
	targetUser *multi.User) error {

	// TODO: read permissions?

	uv := NewUserView(c.Request.Context(), a, sub, targetTenant, targetUser)
	return executeUserTemplate(c, "userRoleEditForm", uv)
}

func handlerAdminUserRoleForm(
	a App,
	c *gin.Context,
	sub *perms.Subject,
	targetTenant *multi.Tenant,
	targetUser *multi.User) error {

	// TODO: read permissions?

	uv := NewUserView(c.Request.Context(), a, sub, targetTenant, targetUser)
	return executeUserTemplate(c, "userRoleForm", uv)
}

type spaceTrimmingRows struct {
	r *csv.Reader
}

func newSpaceTrimmingRows(r *csv.Reader) *spaceTrimmingRows {
	return &spaceTrimmingRows{r: r}
}

// Read reads rows from the underlying CSV reader and trims spaces from each field.
func (s *spaceTrimmingRows) Read() ([]string, error) {
	row, err := s.r.Read()
	if err != nil {
		return nil, err
	}
	for i, field := range row {
		row[i] = strings.TrimSpace(field)
	}
	return row, nil
}

func ifEmpty(s string, def string) string {
	if s == "" {
		return def
	}
	return s
}

func bulkInviteFormError(
	a App, c *gin.Context, sub *perms.Subject, tenant *multi.Tenant, errMsg string) error {

	c.Header("HX-Retarget", "#invite-form")
	tv := NewTenantView(c.Request.Context(), a, sub, tenant).
		WithErrMsg(errMsg)
	return executeTenantTemplate(c, "bulkInviteForm", tv)
}

func handleBulkInviteSubmit(
	a App, c *gin.Context, sub *perms.Subject, targetTenant *multi.Tenant) error {

	// Get the CSV file from a multipart form post
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			return bulkInviteFormError(a, c, sub, targetTenant, "No file specified")
		}
		return bulkInviteFormError(a, c, sub, targetTenant, "Could not process CSV: "+err.Error())
	}
	defer func(file multipart.File) {
		err := file.Close()
		if err != nil {
			zap.L().Error("error closing file", zap.Error(err))
		}
	}(file)

	r := newSpaceTrimmingRows(csv.NewReader(file))

	// Read the header and validate it matches the profile fields
	header, err := r.Read()
	if err != nil {
		return bulkInviteFormError(a, c, sub, targetTenant, "Could not process CSV: "+err.Error())
	}
	allFields := []string{"Email", "Role", "MFA Policy"}
	for _, profileField := range targetTenant.GetUserProfileDefinition().Fields {
		allFields = append(allFields, profileField.UIName)
	}
	if len(header) != len(allFields) {
		return bulkInviteFormError(a, c, sub, targetTenant,
			"CSV header does not match profile fields. Expected: "+strings.Join(allFields, ", "))
	}
	for i, field := range allFields {
		if header[i] != field {
			return bulkInviteFormError(a, c, sub, targetTenant,
				"CSV header does not match profile fields. Expected: "+strings.Join(allFields, ", "))
		}
	}

	// Each row after the header is an invitation
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return bulkInviteFormError(a, c, sub, targetTenant,
				"Failed reading CSV: "+err.Error())
		}
		username := row[0]
		if username == "" {
			// Skip empty rows
			continue
		}

		// Create the invite request from the values in the csv line
		var req rest.InviteRequest
		req.Username = row[0]
		req.Role = multi.RoleSlug(ifEmpty(row[1], multi.RoleGuest))
		req.MfaPolicy = rest.MfaPolicy(ifEmpty(row[2], multi.MFAPolicyNone.String()))
		req.ProfileFields = make([]rest.ProfileField, 0)
		for i := 3; i < len(row); i++ {
			if row[i] == "" {
				// Only set profile fields that have a value
				continue
			}
			req.ProfileFields = append(req.ProfileFields, rest.ProfileField{
				Name:  allFields[i],
				Value: row[i],
			})
		}
		err = rest.HandleInviteToTenant(a, c, sub, targetTenant, req)
		if err != nil {
			return bulkInviteFormError(a, c, sub, targetTenant,
				fmt.Sprintf("Invite failed for %s: %s", req.Username, err.Error()))
		}
	}
	users, err := usersForTenant(c.Request.Context(), a, targetTenant)
	if err != nil {
		return err
	}
	sv := NewAuthenticatedSubjectView(c.Request.Context(), a, sub).
		WithSelectedTen(targetTenant).
		WithUsers(users)
	return executeAdminTemplate(c, "inviteUserTableAndForm", sv)
}

func handleBulkInviteForm(
	a App, c *gin.Context, sub *perms.Subject, targetTenant *multi.Tenant) error {

	_ = sub
	tv := NewTenantView(c.Request.Context(), a, sub, targetTenant)
	return executeTenantTemplate(c, "bulkInviteForm", tv)
}

func handleBulkInviteTemplate(
	a App, c *gin.Context, sub *perms.Subject, targetTenant *multi.Tenant) error {

	_ = a
	_ = sub // TODO: check read permissions here?

	// Create a CSV template file for inviting users that has columns for the Email, Role, and MFA Policy
	// and the profile fields associated with the tenant.
	csvData := [][]string{
		{"Email", "Role", "MFA Policy"},
	}
	for _, profileField := range targetTenant.GetUserProfileDefinition().Fields {
		csvData[0] = append(csvData[0], profileField.UIName)
	}
	b := &bytes.Buffer{}
	w := csv.NewWriter(b)
	err := w.WriteAll(csvData)
	if err != nil {
		return apperr.NewInternalE(err)
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return apperr.NewInternalE(err)
	}
	c.Header("Content-Type", "text/csv")
	filename := fmt.Sprintf(
		"bulk_invite_%s_%s.csv", targetTenant.Name, time.Now().Format("2006-01-02"))
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.String(http.StatusOK, b.String())
	return nil
}
