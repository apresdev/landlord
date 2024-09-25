package rest

import (
	"fmt"

	"github.com/apresdev/landlord/internal/jwx"
	"github.com/apresdev/landlord/internal/multi"
	"github.com/google/uuid"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/apresdev/landlord/internal/db"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
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

	actorTenantID, err := uuid.Parse(tenantID)
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

	t, ok := c.Get(TokenKey)
	if !ok {
		return nil, apperr.NewUnauthorized("no token")
	}
	tok, ok := t.(*jwt.Token)
	if !ok {
		zap.L().Error("could not convert token to *jwt.Token")
		return nil, apperr.NewUnauthorized("no token")
	}
	cid := clientID(c)
	actorGroups, err := jwx.GetClaimStrings(tok, authx.CognitoGroupsClaim)
	if !ok {
		zap.L().Error("could get groups",
			zap.String("clientID", string(cid)),
			zap.Error(err),
			zap.Any("token", tok))
		return nil, apperr.NewUnauthorized("no groups")
	}
	actorRoles, err := authx.ParseRolesFromGroups(actorGroups)
	if err != nil {
		zap.L().Error("could not parse roles from groups",
			zap.String("clientID", string(cid)),
			zap.Error(err),
			zap.Any("groups", actorGroups))
		return nil, apperr.NewUnauthorized("no groups")
	}

	actorTenant, err := db.GetTenantByID(c.Request.Context(), app.DDB(), actorTenantID)
	if err != nil {
		zap.L().Error("error resolving tenant",
			zap.Error(err),
			zap.String("tenantID", tenantID))
		return nil, apperr.NewBadRequest("invalid tenant")
	}
	// TODO: we only allow this operation for members of the root tenant, for now
	// TODO: check if role is able to set user status. Currently all roles within
	// the root tenant are permitted.
	if !authx.IsAuthorizedForTenant(
		c.Request.Context(), app.DDB(), app.RootTenantID(), actorTenant, actorRoles) {

		zap.L().Info("user not authorized for tenant when setting user status",
			zap.String("clientID", string(cid)),
			zap.String("tenantID", tenantID),
			zap.String("roles", actorRoles.String()))
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
