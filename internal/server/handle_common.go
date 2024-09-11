package server

import (
	"github.com/apresdev/landlord/internal/apperr"
	"github.com/apresdev/landlord/internal/multi"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/apresdev/landlord/internal/rest"
	"github.com/golang-jwt/jwt/v5"

	"github.com/apresdev/landlord/internal/db"
	"go.uber.org/zap"

	"github.com/apresdev/landlord/internal/jwx"
	"github.com/gin-gonic/gin"
)

type ClientUser struct {
	CID    jwx.OpaqueClientID
	Tenant *multi.Tenant
	User   *multi.User
}

func resolveClientUser(a *app, c *gin.Context) (ClientUser, error) {
	cid := clientID(c)
	t, ok := c.Get(rest.TokenKey)
	if !ok {
		zap.L().Error("no token found",
			zap.String("clientID", string(cid)))
		return ClientUser{}, apperr.NewUnauthorized("unauthorized")
	}
	tok, ok := t.(*jwt.Token)
	if !ok {
		zap.L().Error("could not convert token to *jwt.Token",
			zap.String("clientID", string(cid)))
		return ClientUser{}, apperr.NewUnauthorized("unauthorized")
	}
	username, err := jwx.GetClaimString(tok, authx.EmailClaim)
	if err != nil {
		zap.L().Error("could not get username from token",
			zap.String("clientID", string(cid)),
			zap.Error(err))
		return ClientUser{}, apperr.NewUnauthorized("unauthorized")
	}
	user, ten, err := db.GetUserAndTenant(c.Request.Context(), a.DDB(), username)
	if err != nil {
		zap.L().Warn("error finding tenant for username",
			zap.String("username", username),
			zap.Error(err))
		return ClientUser{}, apperr.NewUnauthorized("unauthorized")
	}
	return ClientUser{
		CID:    cid,
		Tenant: ten,
		User:   user,
	}, nil
}
