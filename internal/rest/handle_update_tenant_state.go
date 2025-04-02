package rest

import (
	"net/http"

	"github.com/apresdev/landlord/internal/db"
	"github.com/apresdev/landlord/internal/perms"
	"go.uber.org/zap"

	"github.com/apresdev/landlord/internal/multi"

	"github.com/apresdev/landlord/internal/apperr"
	"github.com/apresdev/landlord/internal/authx"
	"github.com/gin-gonic/gin"
)

func handleUpdateTenantState(a App, c *gin.Context) error {
	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		return err
	}
	tenantID := c.Param("tenantID")
	if tenantID == "" {
		return apperr.NewBadRequest("no tenantID provided")
	}
	state := c.Param("tenantState")
	if state == "" {
		return apperr.NewBadRequest("no tenantState provided")
	}
	th, err := multi.ParseTenantHandle(tenantID)
	if err != nil {
		return apperr.NewBadRequest("invalid tenantID")
	}
	targetTenant, err := db.ResolveTenant(c.Request.Context(), a.DBCtx(), th)
	if err != nil {
		return err
	}
	if _, err := HandleUpdateTenantState(a, c, sub, targetTenant, state); err != nil {
		return err
	}
	c.JSON(http.StatusNoContent, nil)
	return nil
}

func HandleUpdateTenantState(
	a App,
	c *gin.Context,
	sub *perms.Subject,
	targetTenant *multi.Tenant,
	state string) (*multi.Tenant, error) {

	resource := perms.NewResource(
		perms.ResourceKindTenantDescription, perms.ResourceNameAny, targetTenant, nil)
	effect, err := perms.Eval(c, a, sub, perms.ActionWrite, resource)
	if err != nil {
		zap.L().Error("could not evaluate permissions", zap.Error(err))
		return nil, apperr.NewUnauthorized("not authorized")
	}
	if effect.IsDenied() {
		zap.L().Info("denied setting state for tenant",
			zap.String("subject", sub.String()),
			zap.String("resource", resource.String()))
		return nil, apperr.NewUnauthorized("not authorized")
	}

	st, err := multi.ParseTenantState(state)
	if err != nil {
		return nil, apperr.NewBadRequest("invalid tenantState")
	}
	ctx := c.Request.Context()
	return authx.SetTenantState(ctx, a.DBCtx(), targetTenant, st)
}
