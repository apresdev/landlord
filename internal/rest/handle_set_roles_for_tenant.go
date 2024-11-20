package rest

import (
	"errors"
	"fmt"
	"slices"

	"github.com/apresdev/landlord/internal/perms"

	"github.com/apresdev/landlord/internal/tinytools"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	cogidptypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"

	"github.com/apresdev/landlord/internal/apperr"
	"github.com/apresdev/landlord/internal/multi"

	"github.com/apresdev/landlord/internal/db"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func handleSetRolesForTenant(app App, c *gin.Context) error {
	req := TenantRolesRequest{}
	if err := c.BindJSON(&req); err != nil {
		return apperr.NewBadRequest(err.Error())
	}
	t := c.Param("tenantID")
	if t == "" {
		return apperr.NewBadRequest("no tenantID")
	}
	return HandleSetRolesForTenant(app, c, req, t)
}

func HandleSetRolesForTenant(app App, c *gin.Context, req TenantRolesRequest, tenantID string) error {
	req.Roles = tinytools.UniqueSlice(req.Roles)
	req.SubTenantRoles = tinytools.UniqueSlice(req.SubTenantRoles)
	if err := validateRoleNames(req.Roles); err != nil {
		return err
	}
	if err := validateRoleNames(req.SubTenantRoles); err != nil {
		return err
	}
	p, err := perms.PrincipalFromContext(c)
	if err != nil {
		return err
	}
	th, err := multi.ParseTenantHandle(tenantID)
	if err != nil {
		return apperr.NewBadRequest(err.Error())
	}
	targetTenant, err := db.ResolveTenant(c.Request.Context(), app.DDB(), th)
	if err != nil {
		return apperr.NewNotFound(err.Error())
	}

	resource := perms.NewResource(
		perms.ResourceKindTenantRoles, perms.ResourceNameAny, targetTenant, nil)
	effect, err := perms.Eval(c, app, p, perms.OperationWrite, resource)
	if err != nil {
		zap.L().Error("could not evaluate permissions", zap.Error(err))
		return apperr.NewUnauthorized("not authorized")
	}
	if effect.IsDenied() {
		zap.L().Info("denied setting roles for tenant",
			zap.String("principal", p.String()),
			zap.String("resource", resource.String()))
		return apperr.NewUnauthorized("not authorized")
	}

	users, err := db.GetUsersForTenant(c.Request.Context(), app.DDB(), targetTenant.ID)
	if err != nil {
		return err
	}
	newRoles := make([]string, 0, len(req.Roles))
	deletedRoles := make([]string, 0, len(targetTenant.Roles))
	for _, r := range req.Roles {
		if slices.Contains(targetTenant.Roles, r) {
			continue
		}
		newRoles = append(newRoles, r)
	}
	for _, r := range targetTenant.Roles {
		// Don't ever delete builtin roles
		if slices.Contains(req.Roles, r) || multi.IsBuiltinRole(r) {
			continue
		}
		deletedRoles = append(deletedRoles, r)
	}
	for _, r := range deletedRoles {
		for _, user := range users {
			if user.Role == r {
				return apperr.NewBadRequest(fmt.Sprintf("role still in use: %s", r))
			}
		}
	}

	for _, r := range newRoles {
		_, err = app.CogIdp().CreateGroup(c.Request.Context(),
			&cognitoidentityprovider.CreateGroupInput{
				GroupName:  aws.String(targetTenant.GroupName(r)),
				UserPoolId: aws.String(targetTenant.UserPoolID),
			})
		var groupExists *cogidptypes.GroupExistsException
		if err != nil && !errors.As(err, &groupExists) {
			zap.L().Error("error creating group for tenant",
				zap.Error(err),
				zap.String("tenant", targetTenant.ID.String()),
				zap.String("role", r))
		}
	}
	for _, r := range deletedRoles {
		_, err = app.CogIdp().DeleteGroup(c.Request.Context(),
			&cognitoidentityprovider.DeleteGroupInput{
				GroupName:  aws.String(targetTenant.GroupName(r)),
				UserPoolId: aws.String(targetTenant.UserPoolID),
			})
		var groupNotFound *cogidptypes.ResourceNotFoundException
		if err != nil && !errors.As(err, &groupNotFound) {
			zap.L().Error("error deleting group for tenant",
				zap.Error(err),
				zap.String("tenant", targetTenant.ID.String()),
				zap.String("role", r))
		}
	}

	targetTenant.Roles = ensureBuiltinNames(req.Roles)
	targetTenant.SubTenantRoles = ensureBuiltinNames(req.SubTenantRoles)

	return db.StoreTenant(c.Request.Context(), app.DDB(), targetTenant)
}

func ensureBuiltinNames(roles []string) []string {
	for _, r := range multi.BuiltInRoles {
		if !slices.Contains(roles, r) {
			roles = append(roles, r)
		}
	}
	return roles
}

func validateRoleNames(roles []string) error {
	for _, r := range roles {
		if err := multi.ValidateRoleName(r); err != nil {
			return err
		}
	}
	return nil
}
