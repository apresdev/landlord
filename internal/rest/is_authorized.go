package rest

import (
	"net/http"
	"strings"

	"github.com/apresdev/landlord/internal/jwx"

	"github.com/apresdev/landlord/internal/db"
	"github.com/apresdev/landlord/internal/multi"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	// TokenKey is the key used to store the token in the gin context.
	TokenKey = "token"
	// TenantIDKey is the key used to store the tenant ID in the gin context.
	TenantIDKey = "tenant-id"
)

func isAuthorized(app App) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid := clientID(c)
		bearer := c.GetHeader("Authorization")
		if bearer == "" {
			zap.L().Info("unauthorized: no header found",
				zap.String("clientID", string(cid)))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		token := strings.TrimSpace(bearer[7:])
		if token == "" {
			zap.L().Info("unauthorized: no token found",
				zap.String("clientID", string(cid)))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		tk := getStr(c, TenantIDKey)
		var ten *multi.Tenant
		if tk != "" {
			tenID, err := uuid.Parse(tk)
			if err != nil {
				zap.L().Error("could not parse tenant ID", zap.Error(err))
				c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
				c.Abort()
				return
			}
			t, err := db.GetTenantByID(c.Request.Context(), app.DDB(), tenID)
			if err != nil {
				zap.L().Info("unauthorized: tenant not found",
					zap.Error(err),
					zap.String("clientID", string(cid)),
					zap.String("tenantID", tenID.String()))
				c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
				c.Abort()
				return
			}
			ten = t
		} else {
			rootTenant, err := db.GetRootTenant(c.Request.Context(), app.DDB())
			if err != nil {
				zap.L().Error("could not get root tenant", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
				c.Abort()
				return
			}
			ten = rootTenant
		}
		up, err := authx.GetUserPoolByNameOrID(
			c.Request.Context(), app.CogIdp(), ten.UserPoolID)
		if err != nil {
			zap.L().Info("unauthorized: could not resolve user pool",
				zap.String("clientID", string(cid)),
				zap.String("tenantID", ten.ID.String()),
				zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		validationURL := jwx.ValidationURL(app.Cfg().Region, aws.ToString(up.Id))
		tok, err := app.Validator().ValidateToken(
			c.Request.Context(), cid, validationURL, token)
		if err != nil {
			zap.L().Info("unauthorized: could not validate token",
				zap.String("clientID", string(cid)),
				zap.String("tenantID", ten.ID.String()),
				zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		c.Set(TokenKey, tok)
		c.Next()
	}
}

func getStr(c *gin.Context, key string) string {
	v, ok := c.Get(key)
	if !ok {
		return ""
	}
	if v2, ok := v.(string); ok {
		return v2
	}
	zap.L().Warn("could not convert context value to string",
		zap.String("key", key),
		zap.Any("value", v))
	return ""
}
