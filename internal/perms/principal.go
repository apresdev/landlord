package perms

import (
	"context"
	"fmt"

	"github.com/google/uuid"

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
	TokenKey = "landlord-token"
	// TenantIDKey is the key used to store the tenant ID in the context.
	TenantIDKey = "landlord-tenant-id"
	// ClientIDKey is the key used to store the client ID in the context.
	ClientIDKey = "landlord-cid"
)

type Subject struct {
	Cid               jwx.OpaqueClientID
	AccessToken       *jwt.Token
	Username          string
	Groups            []string
	Roles             *multi.Roles
	TenantID          uuid.UUID
	SubID             uuid.UUID
	ImpersonateTenant uuid.UUID
	ImpersonateGroup  string
	ImpersonateEmail  string
	ImpersonateSub    uuid.UUID
	user              *multi.User
	tenant            *multi.Tenant
}

func (p *Subject) String() string {
	return fmt.Sprintf("%s@%s", p.Username, p.TenantID.String())
}

func (p *Subject) ResolveTenant(ctx context.Context, app App) (*multi.Tenant, error) {
	if p.tenant != nil {
		return p.tenant, nil
	}
	tenant, err := db.GetTenantByID(ctx, app.DDB(), p.TenantID)
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

func (p *Subject) IsImpersonating() bool {
	return p.ImpersonateTenant != uuid.Nil
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
	if err != nil {
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
	tenantID, err := jwx.GetClaimUUID(tok, authx.CustomTenantClaim)
	if err != nil {
		zap.L().Error("could not parse tenant ID",
			zap.String("clientID", string(cid)),
			zap.Error(err))
		return nil, apperr.NewUnauthorized("not authorized")
	}
	subID, err := jwx.GetClaimUUID(tok, authx.SubClaim)
	if err != nil {
		zap.L().Error("could not get sub claim from token",
			zap.String("clientID", string(cid)),
			zap.Error(err))
		return nil, apperr.NewUnauthorized("not authorized")
	}
	return &Subject{
		Cid:               cid,
		AccessToken:       tok,
		Username:          username,
		Groups:            groups,
		Roles:             roles,
		TenantID:          tenantID,
		SubID:             subID,
		ImpersonateTenant: uuidClaim(tok, authx.CustomImpersonateTenantClaim),
		ImpersonateGroup:  stringClaim(tok, authx.CustomImpersonateGroupClaim),
		ImpersonateEmail:  stringClaim(tok, authx.CustomImpersonateEmailClaim),
		ImpersonateSub:    uuidClaim(tok, authx.CustomImpersonateSubClaim),
	}, nil
}

func uuidClaim(tok *jwt.Token, key string) uuid.UUID {
	id, err := jwx.GetClaimString(tok, key)
	if err != nil {
		if !apperr.IsNotFound(err) {
			zap.L().Error("could not get claim",
				zap.String("key", key),
				zap.Error(err))
		}
		return uuid.Nil
	}
	u, err := uuid.Parse(id)
	if err != nil {
		zap.L().Error("could not parse UUID",
			zap.String("key", key),
			zap.String("value", id),
			zap.Error(err))
		return uuid.Nil
	}
	return u
}

func stringClaim(tok *jwt.Token, key string) string {
	v, err := jwx.GetClaimString(tok, key)
	if err != nil {
		if !apperr.IsNotFound(err) {
			zap.L().Error("could not get claim",
				zap.String("key", key),
				zap.Error(err))
		}
		return ""
	}
	return v
}
