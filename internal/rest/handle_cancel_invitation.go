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

func handleCancelInvitation(app App, c *gin.Context) error {
	username := c.Param("username")
	if username == "" {
		return apperr.NewBadRequest("no username provided")
	}
	return HandleCancelInvitation(app, c, username)
}

// TODO: change this API to use a User UUID
func HandleCancelInvitation(app App, c *gin.Context, username string) error {
	t, ok := c.Get(TokenKey)
	if !ok {
		return apperr.NewUnauthorized("no token")
	}
	tok, ok := t.(*jwt.Token)
	if !ok {
		zap.L().Error("could not convert token to *jwt.Token")
		return apperr.NewUnauthorized("no token")
	}
	cid := clientID(c)
	groups, err := jwx.GetClaimStrings(tok, authx.CognitoGroupsClaim)
	if !ok {
		zap.L().Error("could get groups",
			zap.String("clientID", string(cid)),
			zap.Error(err),
			zap.Any("token", tok))
		return apperr.NewUnauthorized("no groups")
	}
	roles, err := authx.ParseRolesFromGroups(groups)
	if err != nil {
		zap.L().Error("could not parse roles from groups",
			zap.String("clientID", string(cid)),
			zap.Error(err),
			zap.Any("groups", groups))
		return apperr.NewUnauthorized("no groups")
	}
	tenant, err := jwx.GetClaimString(tok, authx.CustomTenantClaim)
	if err != nil {
		zap.L().Error("could not get tenant claim",
			zap.String("clientID", string(cid)),
			zap.Error(err),
			zap.Any("token", tok))
		return apperr.NewUnauthorized("no tenant")
	}
	th, err := multi.ParseTenantHandle(tenant)
	if err != nil {
		zap.L().Error("error parsing tenant handle",
			zap.Error(err),
			zap.String("tenantID", tenant))
		return apperr.NewUnauthorized("invalid tenant")
	}
	ten, err := db.ResolveTenant(c.Request.Context(), app.DDB(), th)
	if err != nil {
		zap.L().Error("error resolving tenant",
			zap.Error(err),
			zap.String("tenantID", th.String()))
		return apperr.NewBadRequest("invalid tenant")
	}
	if !authx.IsAuthorizedForTenant(
		c.Request.Context(), app.DDB(), app.RootTenantID(), ten, roles) {
		return apperr.NewUnauthorized("not authorized")
	}

	// TODO: check if the caller's role is allowed to cancel invitations

	return authx.CancelInvitation(
		c.Request.Context(), app.DDB(), app.CogIdp(), username)
}
