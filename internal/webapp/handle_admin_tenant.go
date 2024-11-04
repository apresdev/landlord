package webapp

import (
	"fmt"
	"slices"
	"sort"

	"github.com/apresdev/landlord/internal/authx"

	"github.com/apresdev/landlord/internal/rest"
	"github.com/apresdev/landlord/internal/tinytools"
	"go.uber.org/zap"

	"github.com/apresdev/landlord/internal/apperr"
	"github.com/apresdev/landlord/internal/multi"

	"github.com/apresdev/landlord/internal/db"
	"github.com/gin-gonic/gin"
)

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

	render(c, tenantPage(a.Settings().AppName(), cu, ten))

	return nil
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
		render(c, tenantSummaryTab(
			tab, a.Settings().AppName(), cu, ten, cognitoURL, users, rolesWithCount(ten.Roles, users)))
		return nil
	} else if tab == "users" {
		render(c, tenantUsersTab(
			tab, a.Settings().AppName(), cu, ten, users, rolesWithCount(ten.Roles, users)))
		return nil
	} else if tab == "roles" {
		render(c, tenantRolesTab(
			tab, a.Settings().AppName(), cu, ten, users, rolesWithCount(ten.Roles, users)))
		return nil
	} else {
		zap.L().Info("unknown tenant tab", zap.String("tab", tab))
	}
	return nil
}

func handleAdminAddRole(a App, c *gin.Context) error {
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
		render(c, tenantAddRoleTableAndForm(ten, rolesWithCount(ten.Roles, users),
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
		render(c, tenantAddRoleTableAndForm(ten, rolesWithCount(req.Roles, users), ""))
	}
	return nil
}

func handleAdminDeleteRole(a App, c *gin.Context) error {
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
	render(c, tenantAddRoleTableAndForm(ten, rolesWithCount(req.Roles, users), ""))
	return nil
}

func handleInviteUserForm(a App, c *gin.Context) error {
	tenantID := c.Param("tenantID")
	th, err := multi.ParseTenantHandle(tenantID)
	if err != nil {
		return apperr.NewBadRequestE(err)
	}
	ten, err := db.ResolveTenant(c.Request.Context(), a.DDB(), th)
	if err != nil {
		return apperr.NewNotFoundE(err)
	}
	render(c, tenantInviteUserForm(tenantID, ten, multi.MFAPolicyStrings()))
	return nil
}

func handleInviteUser(a App, c *gin.Context) error {
	var req rest.InviteRequest
	req.Username = c.PostForm("username")
	req.Tenant = c.Param("tenantID")
	req.Role = c.PostForm("role")
	req.MfaPolicy = rest.MfaPolicy(c.PostForm("mfaPolicy"))
	if err := rest.HandleInviteToTenant(a, c, req); err != nil {
		return err
	}
	th, err := multi.ParseTenantHandle(req.Tenant)
	if err != nil {
		return apperr.NewBadRequestE(err)
	}
	ten, err := db.ResolveTenant(c.Request.Context(), a.DDB(), th)
	if err != nil {
		return apperr.NewNotFoundE(err)
	}
	users, err := db.GetUsersForTenant(c.Request.Context(), a.DDB(), ten.ID)
	if err != nil {
		return apperr.NewInternalE(err)
	}
	render(c, tenantInviteUserTableAndForm(ten, users, ""))
	return nil
}
