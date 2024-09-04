package rest

import (
	"github.com/apresdev/landlord/internal/apperr"
	"github.com/apresdev/landlord/internal/authx"
	"github.com/apresdev/landlord/internal/jwx"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

type UserResolution struct {
	Username    string
	Tenant      string
	Roles       authx.Roles
	ClientID    jwx.OpaqueClientID
	AccessToken string
}

func resolveUser(c *gin.Context) (*UserResolution, error) {
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
		zap.L().Error("could not get groups",
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
	username, err := jwx.GetClaimString(tok, authx.EmailClaim)
	if err != nil {
		zap.L().Error("could not get username",
			zap.String("clientID", string(cid)),
			zap.Error(err),
			zap.Any("token", tok))
		return nil, apperr.NewUnauthorized("no username")
	}
	tenant, err := jwx.GetClaimString(tok, authx.CustomTenantClaim)
	if err != nil {
		zap.L().Error("could not get tenant",
			zap.String("clientID", string(cid)),
			zap.Error(err),
			zap.Any("token", tok))
		return nil, apperr.NewUnauthorized("no tenant")
	}
	return &UserResolution{
		Username:    username,
		Tenant:      tenant,
		Roles:       roles,
		ClientID:    cid,
		AccessToken: tok.Raw,
	}, nil
}
