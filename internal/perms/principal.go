package perms

import (
	"context"
	"fmt"

	"github.com/apresdev/landlord/internal/db"

	"github.com/apresdev/landlord/internal/apperr"
	"github.com/apresdev/landlord/internal/authx"
	"github.com/apresdev/landlord/internal/jwx"
	"github.com/apresdev/landlord/internal/multi"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

const (
	// TokenKey is the key used to store the token in the context.
	TokenKey = "token"
	// TenantIDKey is the key used to store the tenant ID in the context.
	TenantIDKey = "tenant-id"
	// ClientIDKey is the key used to store the client ID in the context.
	ClientIDKey = "cid"
)

type Subject struct {
	Cid         jwx.OpaqueClientID
	AccessToken *jwt.Token
	Username    string
	Groups      []string
	Roles       *multi.Roles
	Th          multi.TenantHandle
	user        *multi.User
	tenant      *multi.Tenant
}

func (p *Subject) String() string {
	return fmt.Sprintf("%s@%s", p.Username, p.Th)
}

func (p *Subject) ResolveTenant(ctx context.Context, app App) (*multi.Tenant, error) {
	if p.tenant != nil {
		return p.tenant, nil
	}
	tenant, err := db.ResolveTenant(ctx, app.DDB(), p.Th)
	if err != nil {
		return nil, err
	}
	p.tenant = tenant
	return tenant, nil
}

func (p *Subject) ResolveUser(ctx context.Context, app App) (*multi.User, error) {
	if p.user != nil {
		return p.user, nil
	}
	user, err := db.GetUserByName(ctx, app.DDB(), p.Username)
	if err != nil {
		return nil, err
	}
	p.user = user
	return user, nil
}

func ClientID(c *gin.Context) jwx.OpaqueClientID {
	return jwx.OpaqueClientID(c.ClientIP())
}

func SubjectFromContext(ctx context.Context) (*Subject, error) {
	t := ctx.Value(TokenKey)
	if t == nil {
		return nil, apperr.NewUnauthorized("no token")
	}
	tok, ok := t.(*jwt.Token)
	if !ok {
		zap.L().Error("could not convert token to *jwt.Token")
		return nil, apperr.NewUnauthorized("no token")
	}

	// We do not fail if the client ID is not present. It's only for
	// logging / troubleshooting purposes.
	c := ctx.Value(ClientIDKey)
	cid := jwx.OpaqueClientID("unknown")
	if c != nil {
		if v, ok := c.(jwx.OpaqueClientID); ok {
			cid = v
		}
	}

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
	return &Subject{
		Cid:         cid,
		AccessToken: tok,
		Username:    username,
		Groups:      groups,
		Roles:       roles,
		Th:          th,
	}, nil
}
