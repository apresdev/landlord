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

func handleUpdateRolesForTenant(a App, c *gin.Context) error {
	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		return err
	}
	tenantID := c.Param("tenantID")
	if tenantID == "" {
		return apperr.NewBadRequest("no tenantID")
	}
	th, err := multi.ParseTenantHandle(tenantID)
	if err != nil {
		return apperr.NewBadRequest(err.Error())
	}
	targetTenant, err := db.ResolveTenant(c.Request.Context(), a.DBCtx(), th)
	if err != nil {
		return apperr.NewNotFound(err.Error())
	}
	req := TenantRolesRequest{}
	if err := c.BindJSON(&req); err != nil {
		return apperr.NewBadRequest(err.Error())
	}
	_, err = HandleUpdateRolesForTenant(a, c, sub, targetTenant, req)
	return err
}

func HandleUpdateRolesForTenant(
	a App,
	c *gin.Context,
	sub *perms.Subject,
	targetTenant *multi.Tenant,
	req TenantRolesRequest) (*multi.Tenant, error) {

	req.Roles = tinytools.UniqueSlice(req.Roles)
	req.SubTenantRoles = tinytools.UniqueSlice(req.SubTenantRoles)
	if err := validateRoleNames(req.Roles); err != nil {
		return nil, err
	}
	if err := validateRoleNames(req.SubTenantRoles); err != nil {
		return nil, err
	}

	resource := perms.NewResource(
		perms.ResourceKindTenantRoles, perms.ResourceNameAny, targetTenant, nil)
	effect, err := perms.Eval(c, a, sub, perms.ActionWrite, resource)
	if err != nil {
		zap.L().Error("could not evaluate permissions", zap.Error(err))
		return nil, apperr.NewUnauthorized("not authorized")
	}
	if effect.IsDenied() {
		zap.L().Info("denied setting roles for tenant",
			zap.String("subject", sub.String()),
			zap.String("resource", resource.String()))
		return nil, apperr.NewUnauthorized("not authorized")
	}

	users, err := db.GetUsersForTenant(c.Request.Context(), a.DBCtx(), targetTenant.ID)
	if err != nil {
		return nil, err
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
				return nil, apperr.NewBadRequest(fmt.Sprintf("role still in use: %s", r))
			}
		}
	}

	for _, r := range newRoles {
		_, err = a.CogIdp().CreateGroup(c.Request.Context(),
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
		_, err = a.CogIdp().DeleteGroup(c.Request.Context(),
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

	err = db.StoreTenant(c.Request.Context(), a.DBCtx(), targetTenant)
	if err != nil {
		return nil, err
	}

	return targetTenant, err
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
