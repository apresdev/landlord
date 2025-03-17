package rest

import (
	"net/http"

	"github.com/apresdev/landlord/internal/multi"
	"github.com/google/uuid"

	"github.com/apresdev/landlord/internal/perms"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/apresdev/landlord/internal/db"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func handleUpdateUserRole(app App, c *gin.Context) error {
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
	_, _, err = HandleUpdateUserRole(app, c, sub, req.Role, tenantID, userID)
	if err != nil {
		return err
	}
	c.JSON(http.StatusNoContent, nil)
	return nil
}

func HandleUpdateUserRole(
	app App,
	c *gin.Context,
	sub *perms.Subject,
	role string,
	tenantID, userID uuid.UUID) (*multi.Tenant, *multi.User, error) {

	targetTenant, err := db.GetTenantByID(c.Request.Context(), app.DDB(), tenantID)
	if err != nil {
		zap.L().Error("error fetching tenant",
			zap.Error(err),
			zap.String("tenantID", tenantID.String()))
		return nil, nil, apperr.NewUnauthorized("not authorized")
	}
	targetUser, err := db.GetUserInTenantByID(c.Request.Context(), app.DDB(), tenantID, userID)
	if err != nil {
		zap.L().Error("error fetching user",
			zap.Error(err),
			zap.String("tenantID", tenantID.String()),
			zap.String("userID", userID.String()))
		return nil, nil, apperr.NewUnauthorized("not authorized")
	}

	resource := perms.NewResource(
		perms.ResourceKindUserRole, role, targetTenant, targetUser)
	effect, err := perms.Eval(c, app, sub, perms.ActionWrite, resource)
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
	err = db.StoreUser(c.Request.Context(), app.DDB(), targetUser)
	if err != nil {
		zap.L().Error("error updating user",
			zap.String("tenantID", tenantID.String()),
			zap.String("userID", userID.String()),
			zap.Error(err))
		return nil, nil, apperr.NewInternal("error updating user")
	}

	return targetTenant, targetUser, nil
}
