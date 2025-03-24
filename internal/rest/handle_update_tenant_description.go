package rest

import (
	"net/http"

	"github.com/apresdev/landlord/internal/perms"
	"go.uber.org/zap"

	"github.com/apresdev/landlord/internal/db"

	"github.com/apresdev/landlord/internal/multi"

	"github.com/apresdev/landlord/internal/apperr"
	"github.com/gin-gonic/gin"
)

func handleUpdateTenantDescription(app App, c *gin.Context) error {
	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		return err
	}
	tenantID := c.Param("tenantID")
	if tenantID == "" {
		return apperr.NewBadRequest("no tenantID provided")
	}
	th, err := multi.ParseTenantHandle(tenantID)
	if err != nil {
		return apperr.NewBadRequest("invalid tenantID")
	}
	targetTenant, err := db.ResolveTenant(c.Request.Context(), app.DDB(), th)
	if err != nil {
		return err
	}
	var req TenantSetDescriptionRequest
	if err := c.BindJSON(&req); err != nil {
		return apperr.NewBadRequest("invalid request")
	}
	if _, err := HandleUpdateTenantDescription(
		app, c, sub, targetTenant, req.Description); err != nil {

		return err
	}
	c.JSON(http.StatusNoContent, nil)
	return nil
}

func HandleUpdateTenantDescription(
	app App,
	c *gin.Context,
	sub *perms.Subject,
	targetTenant *multi.Tenant,
	description string) (*multi.Tenant, error) {

	if err := multi.ValidateTenantDescription(description); err != nil {
		return nil, err
	}

	resource := perms.NewResource(
		perms.ResourceKindTenantDescription, perms.ResourceNameAny, targetTenant, nil)
	effect, err := perms.Eval(c, app, sub, perms.ActionWrite, resource)
	if err != nil {
		zap.L().Error("could not evaluate permissions", zap.Error(err))
		return nil, apperr.NewUnauthorized("not authorized")
	}
	if effect.IsDenied() {
		zap.L().Info("denied setting description for tenant",
			zap.String("subject", sub.String()),
			zap.String("resource", resource.String()))
		return nil, apperr.NewUnauthorized("not authorized")
	}

	targetTenant.Description = description
	err = db.StoreTenant(c.Request.Context(), app.DDB(), targetTenant)
	if err != nil {
		return nil, err
	}
	return targetTenant, nil
}
