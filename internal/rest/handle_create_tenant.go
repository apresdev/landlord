package rest

import (
	"net/http"

	"github.com/apresdev/landlord/internal/perms"
	"go.uber.org/zap"

	"github.com/apresdev/landlord/internal/apperr"
	"github.com/apresdev/landlord/internal/authx"
	"github.com/gin-gonic/gin"
)

func handleCreateTenant(a App, c *gin.Context) error {
	var req CreateTenantRequest
	if err := c.BindJSON(&req); err != nil {
		return apperr.NewBadRequest(err.Error())
	}
	resp, err := HandleCreateTenant(a, c, req)
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, resp)
	return nil
}

func HandleCreateTenant(
	a App, c *gin.Context, req CreateTenantRequest) (
	*TenantResponse, error) {

	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		return nil, err
	}

	targetTenant, err := sub.ResolveTenant(c.Request.Context(), a)
	if err != nil {
		zap.L().Error("could not resolve subject tenant", zap.Error(err))
		return nil, apperr.NewUnauthorized("not authorized")
	}
	targetUser, err := sub.ResolveUser(c.Request.Context(), a)
	if err != nil {
		zap.L().Error("could not resolve subject user", zap.Error(err))
		return nil, apperr.NewUnauthorized("not authorized")
	}

	// When creating a tenant, the resource owner is the parent tenant. In this
	// case, the parent tenant is the one associated with the subject.
	resource := perms.NewResource(
		perms.ResourceKindTenant, req.Name, targetTenant, targetUser)
	effect, err := perms.Eval(c, a, sub, perms.ActionWrite, resource)
	if err != nil {
		zap.L().Error("could not evaluate permissions", zap.Error(err))
		return nil, apperr.NewUnauthorized("not authorized")
	}
	if effect.IsDenied() {
		zap.L().Info("denied creating tenant",
			zap.String("subject", sub.String()),
			zap.String("resource", resource.String()))
		return nil, apperr.NewUnauthorized("not authorized")
	}

	tenant, err := authx.CreateTenant(
		c.Request.Context(),
		a.DBCtx(),
		a.RootTenantID(),
		a.CogIdp(),
		req.Name,
		req.Description)
	if err != nil {
		return nil, err
	}
	var resp TenantResponse
	resp.Tenant.Id = tenant.ID.String()
	resp.Tenant.Name = tenant.Name
	resp.Tenant.Description = tenant.Description
	resp.Tenant.Roles = tenant.Roles
	return &resp, nil
}
