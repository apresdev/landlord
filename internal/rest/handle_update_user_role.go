package rest

import (
	"net/http"

	"github.com/apresdev/landlord/internal/multi"
	"github.com/apresdev/landlord/internal/perms"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/apresdev/landlord/internal/db"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func handleUpdateUserRole(a App, c *gin.Context) error {
	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		return apperr.NewUnauthorized("no token")
	}
	var req UpdateUserRoleRequest
	if err := c.BindJSON(&req); err != nil {
		return apperr.NewBadRequest(err.Error())
	}
	tenantID, err := UUIDParam(c, "tenantID")
	if err != nil {
		return apperr.NewBadRequest("invalid tenant ID")
	}
	userID, err := UUIDParam(c, "userID")
	if err != nil {
		return apperr.NewBadRequest("invalid user ID")
	}
	targetTenant, err := db.GetTenantByID(c.Request.Context(), a.DBCtx(), tenantID)
	if err != nil {
		zap.L().Error("error fetching tenant",
			zap.Error(err),
			zap.String("tenantID", tenantID.String()))
		return apperr.NewUnauthorized("not authorized")
	}
	targetUser, err := db.GetUserInTenantByID(c.Request.Context(), a.DBCtx(), tenantID, userID)
	if err != nil {
		zap.L().Error("error fetching user",
			zap.Error(err),
			zap.String("tenantID", tenantID.String()),
			zap.String("userID", userID.String()))
		return apperr.NewUnauthorized("not authorized")
	}
	_, _, err = HandleUpdateUserRole(a, c, sub, targetTenant, targetUser, req.Role)
	if err != nil {
		return err
	}
	c.JSON(http.StatusNoContent, nil)
	return nil
}

func HandleUpdateUserRole(
	a App,
	c *gin.Context,
	sub *perms.Subject,
	targetTenant *multi.Tenant,
	targetUser *multi.User,
	role string) (*multi.Tenant, *multi.User, error) {

	resource := perms.NewResource(
		perms.ResourceKindUserRole, role, targetTenant, targetUser)
	effect, err := perms.Eval(c, a, sub, perms.ActionWrite, resource)
	if err != nil {
		zap.L().Error("could not evaluate permissions", zap.Error(err))
		return nil, nil, apperr.NewUnauthorized("not authorized")
	}
	if effect.IsDenied() {
		zap.L().Info("denied setting role for user",
			zap.String("subject", sub.String()),
			zap.String("resource", resource.String()))
		return nil, nil, apperr.NewUnauthorized("not authorized")
	}

	targetUser.Role = role
	err = db.StoreUser(c.Request.Context(), a.DBCtx(), targetUser)
	if err != nil {
		zap.L().Error("error updating user",
			zap.String("tenantID", targetTenant.ID.String()),
			zap.String("userID", targetUser.ID.String()),
			zap.Error(err))
		return nil, nil, apperr.NewInternal("error updating user")
	}

	return targetTenant, targetUser, nil
}
