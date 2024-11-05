package rest

import (
	"fmt"

	"github.com/apresdev/landlord/internal/multi"
	"github.com/google/uuid"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/apresdev/landlord/internal/db"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func handleSetUserStatus(app App, c *gin.Context) error {
	tenantID := c.Param("tenantID")
	if tenantID == "" {
		return apperr.NewBadRequest("no tenantID provided")
	}
	username := c.Param("username")
	if username == "" {
		return apperr.NewBadRequest("no username provided")
	}
	s := c.Param("userStatus")
	if s == "" {
		return apperr.NewBadRequest("no userStatus provided")
	}
	_, err := HandleSetUserStatus(app, c, tenantID, username, s)
	return err
}

// TODO: change this API to use a User UUID
func HandleSetUserStatus(
	app App, c *gin.Context, tenantID, username, rawTargetStatus string) (
	*multi.User, error) {

	targetTenantID, err := uuid.Parse(tenantID)
	if err != nil {
		zap.L().Error("error parsing tenant handle",
			zap.Error(err),
			zap.String("tenantID", tenantID))
		return nil, apperr.NewBadRequest("invalid tenantID")
	}
	targetStatus, err := multi.ParseUserStatus(rawTargetStatus)
	if err != nil {
		return nil, apperr.NewBadRequestE(err)
	}

	rc, err := authx.RequesterFromContext(c)
	if err != nil {
		return nil, err
	}

	targetTenant, err := db.GetTenantByID(c.Request.Context(), app.DDB(), targetTenantID)
	if err != nil {
		zap.L().Error("error resolving tenant",
			zap.Error(err),
			zap.String("tenantID", tenantID))
		return nil, apperr.NewBadRequest("invalid tenant")
	}
	// TODO: we only allow this operation for members of the root tenant, for now
	// TODO: check if role is able to set user status. Currently all roles within
	// the root tenant are permitted.
	if !authx.IsAuthorizedForTenant(app.RootTenantID(), targetTenant, rc.Roles) {
		zap.L().Info("user not authorized for tenant when setting user status",
			zap.String("clientID", string(rc.Cid)),
			zap.String("tenantID", tenantID),
			zap.String("roles", rc.Roles.String()))
		return nil, apperr.NewUnauthorized("not authorized")
	}

	targetUser, err := db.GetUserByName(c.Request.Context(), app.DDB(), username)
	if err != nil {
		zap.L().Info("user not found when setting user status",
			zap.String("username", username),
			zap.Error(err))
		return nil, apperr.NewNotFound(fmt.Sprintf("user not found: %s", username))
	}
	u, err := db.SetUserStatus(c.Request.Context(), app.DDB(), targetUser, targetStatus)
	if err != nil {
		return nil, err
	}

	return u, nil
}
