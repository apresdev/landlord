package authx

import (
	"github.com/apresdev/landlord/internal/apperr"
	"github.com/apresdev/landlord/internal/jwx"
	"github.com/apresdev/landlord/internal/multi"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

const (
	// TokenKey is the key used to store the token in the gin context.
	TokenKey = "token"
	// TenantIDKey is the key used to store the tenant ID in the gin context.
	TenantIDKey = "tenant-id"
)

type Requester struct {
	Cid         jwx.OpaqueClientID
	AccessToken *jwt.Token
	Username    string
	Groups      []string
	Roles       Roles
	Th          multi.TenantHandle
	User        *multi.User
	Tenant      *multi.Tenant
}

func ClientID(c *gin.Context) jwx.OpaqueClientID {
	return jwx.OpaqueClientID(c.ClientIP())
}

func RequesterFromContext(c *gin.Context) (*Requester, error) {
	t, ok := c.Get(TokenKey)
	if !ok {
		return nil, apperr.NewUnauthorized("no token")
	}
	tok, ok := t.(*jwt.Token)
	if !ok {
		zap.L().Error("could not convert token to *jwt.Token")
		return nil, apperr.NewUnauthorized("no token")
	}
	cid := ClientID(c)
	groups, err := jwx.GetClaimStrings(tok, CognitoGroupsClaim)
	if !ok {
		zap.L().Error("could get groups",
			zap.String("clientID", string(cid)),
			zap.Error(err),
			zap.Any("token", tok))
		return nil, apperr.NewUnauthorized("no groups")
	}
	roles, err := ParseRolesFromGroups(groups)
	if err != nil {
		zap.L().Error("could not parse roles from groups",
			zap.String("clientID", string(cid)),
			zap.Error(err),
			zap.Any("groups", groups))
		return nil, apperr.NewUnauthorized("no groups")
	}
	username, err := jwx.GetClaimString(tok, EmailClaim)
	if err != nil {
		zap.L().Error("could not get username from token",
			zap.String("clientID", string(cid)),
			zap.Error(err))
		return nil, apperr.NewUnauthorized("not authorized")
	}
	tenant, err := jwx.GetClaimString(tok, CustomTenantClaim)
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
	return &Requester{
		Cid:         cid,
		AccessToken: tok,
		Username:    username,
		Groups:      groups,
		Roles:       roles,
		Th:          th,
	}, nil
}
