package rest

import (
	"net/http"

	"github.com/apresdev/landlord/internal/perms"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func handleClearImpersonation(app App, c *gin.Context) error {
	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		return err
	}
	err = HandleClearImpersonation(app, c, sub)
	if err != nil {
		return err
	}
	c.JSON(http.StatusNoContent, nil)
	return nil
}

func HandleClearImpersonation(app App, c *gin.Context, sub *perms.Subject) error {
	// The target user and tenant of this API are from the subject
	targetUser, err := sub.ResolveUser(c.Request.Context(), app)
	if err != nil {
		zap.L().Error("could not resolve subject user", zap.Error(err))
		return apperr.NewUnauthorized("not authorized")
	}
	targetTenant, err := sub.ResolveTenant(c.Request.Context(), app)
	if err != nil {
		zap.L().Error("could not resolve subject tenant", zap.Error(err))
		return apperr.NewUnauthorized("not authorized")
	}

	resource := perms.NewResource(
		perms.ResourceKindImpersonation, targetUser.Username, targetTenant, targetUser)
	effect, err := perms.Eval(c, app, sub, perms.ActionDelete, resource)
	if err != nil {
		zap.L().Error("could not evaluate permissions", zap.Error(err))
		return apperr.NewUnauthorized("not authorized")
	}
	if effect.IsDenied() {
		zap.L().Info("denied clearing impersonation",
			zap.String("subject", sub.String()),
			zap.String("resource", resource.String()))
		return apperr.NewUnauthorized("not authorized")
	}

	err = authx.ClearImpersonate(
		c.Request.Context(), app.CogIdp(), targetTenant.UserPoolID, targetUser.Username)
	if err != nil {
		zap.L().Error("could not clear impersonation",
			zap.String("clientID", string(sub.Cid)),
			zap.Error(err))
		return apperr.NewUnauthorized("not authorized")
	}
	return nil
}
