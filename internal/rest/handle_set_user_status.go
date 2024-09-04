package rest

import (
	"github.com/apresdev/landlord/internal/jwx"
	"github.com/apresdev/landlord/internal/multi"

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

// TODO: this shouldn't allow for the activation of a user that is "invited"
func HandleSetUserStatus(
	app App, c *gin.Context, tenantID, username, rawTargetStatus string) (
	*multi.User, error) {

	th, err := multi.ParseTenantHandle(tenantID)
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
	groups, err := jwx.GetClaimStrings(tok, authx.CognitoGroupsClaim)
	if !ok {
		zap.L().Error("could get groups",
			zap.String("clientID", string(cid)),
			zap.Error(err),
			zap.Any("token", tok))
		return nil, apperr.NewUnauthorized("no groups")
	}
	roles, err := authx.ParseRolesFromGroups(groups)
	if err != nil {
		zap.L().Error("could not parse roles from groups",
			zap.String("clientID", string(cid)),
			zap.Error(err),
			zap.Any("groups", groups))
		return nil, apperr.NewUnauthorized("no groups")
	}

	ten, err := db.ResolveTenant(c.Request.Context(), app.DDB(), th)
	if err != nil {
		zap.L().Error("error resolving tenant",
			zap.Error(err),
			zap.String("tenantID", tenantID))
		return nil, apperr.NewBadRequest("invalid tenant")
	}
	if !authx.IsAuthorizedForTenant(c.Request.Context(), app.DDB(), ten, roles) {
		zap.L().Info("user not authorized for tenant",
			zap.String("clientID", string(cid)),
			zap.String("tenantID", tenantID),
			zap.String("roles", roles.String()))
		return nil, apperr.NewUnauthorized("not authorized")
	}

	// TODO: check if role is able to set user status. Currently all
	// roles are allowed to set status

	u, err := db.SetUserStatus(
		c.Request.Context(), app.DDB(), ten.ID, username, targetStatus)
	if err != nil {
		return nil, err
	}
	return u, nil
}
