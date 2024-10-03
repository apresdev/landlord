package rest

import (
	"github.com/apresdev/landlord/internal/apperr"
	"github.com/apresdev/landlord/internal/authx"
	"github.com/apresdev/landlord/internal/jwx"
	"github.com/apresdev/landlord/internal/multi"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

type requesterCtx struct {
	cid         jwx.OpaqueClientID
	accessToken *jwt.Token
	username    string
	groups      []string
	roles       authx.Roles
	th          multi.TenantHandle
}

func requesterContext(c *gin.Context) (*requesterCtx, error) {
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
	username, err := jwx.GetClaimString(tok, authx.EmailClaim)
	if err != nil {
		zap.L().Error("could not get username from token",
			zap.String("clientID", string(cid)),
			zap.Error(err))
		return nil, apperr.NewUnauthorized("not authorized")
	}
	tenant, err := jwx.GetClaimString(tok, authx.CustomTenantClaim)
	if err != nil {
		zap.L().Error("could not get tenant claim",
			zap.String("clientID", string(cid)),
			zap.Error(err),
			zap.Any("token", tok))
		return nil, apperr.NewUnauthorized("no tenant")
	}
	th, err := multi.ParseTenantHandle(tenant)
	if err != nil {
		zap.L().Error("error parsing tenant handle",
			zap.Error(err),
			zap.String("tenantID", tenant))
		return nil, apperr.NewUnauthorized("invalid tenant")
	}
	return &requesterCtx{
		cid:         cid,
		accessToken: tok,
		username:    username,
		groups:      groups,
		roles:       roles,
		th:          th,
	}, nil
}
