package rest

import (
	"fmt"
	"net/http"

	"github.com/apresdev/landlord/internal/perms"
	"go.uber.org/zap"

	"github.com/apresdev/landlord/internal/apperr"
	"github.com/apresdev/landlord/internal/authx"
	"github.com/gin-gonic/gin"
)

func handleCreateTenant(app App, c *gin.Context) error {
	var req CreateTenantRequest
	if err := c.BindJSON(&req); err != nil {
		return apperr.NewBadRequest(err.Error())
	}
	resp, err := HandleCreateTenant(app, c, req)
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, resp)
	return nil
}

func HandleCreateTenant(
	app App, c *gin.Context, req CreateTenantRequest) (
	*TenantResponse, error) {

	p, err := perms.PrincipalFromContext(c)
	if err != nil {
		return nil, err
	}

	targetTenant, err := p.ResolveTenant(c.Request.Context(), app)
	if err != nil {
		zap.L().Error("could not resolve principal tenant", zap.Error(err))
		return nil, apperr.NewUnauthorized("not authorized")
	}
	targetUser, err := p.ResolveUser(c.Request.Context(), app)
	if err != nil {
		zap.L().Error("could not resolve principal user", zap.Error(err))
		return nil, apperr.NewUnauthorized("not authorized")
	}

	// When creating a tenant, the resource owner is the parent tenant. In this
	// case, the parent tenant is the one associated with the principal.
	resource := perms.NewResource(
		perms.ResourceKindTenant, req.Name, targetTenant, targetUser)
	effect, err := perms.Eval(c, app, p, perms.OperationWrite, resource)
	if err != nil {
		zap.L().Error("could not evaluate permissions", zap.Error(err))
		return nil, apperr.NewUnauthorized("not authorized")
	}
	if effect.IsDenied() {
		zap.L().Info("denied clearing impersonation",
			zap.String("principal", p.String()),
			zap.String("resource", resource.String()))
		return nil, apperr.NewUnauthorized("not authorized")
	}

	tenant, err := authx.CreateTenant(
		c.Request.Context(),
		app.DDB(),
		app.RootTenantID(),
		app.CogIdp(),
		req.Name,
		req.Description)
	if err != nil {
		return nil, fmt.Errorf("error creating tenant: %w", err)
	}
	var resp TenantResponse
	resp.Tenant.Id = tenant.ID.String()
	resp.Tenant.Name = tenant.Name
	resp.Tenant.Description = tenant.Description
	resp.Tenant.Roles = tenant.Roles
	return &resp, nil
}
