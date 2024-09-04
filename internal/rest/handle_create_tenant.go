package rest

import (
	"fmt"
	"net/http"

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

	ddb := app.DDB()
	cogIdp := app.CogIdp()
	ctx := c.Request.Context()
	tenant, err := authx.CreateTenant(ctx, ddb, cogIdp, req.Name, req.Description)
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
