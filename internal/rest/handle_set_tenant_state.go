package rest

import (
	"net/http"

	"github.com/apresdev/landlord/internal/multi"

	"github.com/apresdev/landlord/internal/apperr"
	"github.com/apresdev/landlord/internal/authx"
	"github.com/gin-gonic/gin"
)

func handleSetTenantState(app App, c *gin.Context) error {
	id := c.Param("tenantID")
	if id == "" {
		return apperr.NewBadRequest("no tenantID provided")
	}
	state := c.Param("tenantState")
	if state == "" {
		return apperr.NewBadRequest("no tenantState provided")
	}
	if _, err := HandleSetTenantState(app, c, id, state); err != nil {
		return err
	}
	c.JSON(http.StatusNoContent, nil)
	return nil
}

func HandleSetTenantState(app App, c *gin.Context, tenantID, state string) (*multi.Tenant, error) {
	th, err := multi.ParseTenantHandle(tenantID)
	if err != nil {
		return nil, apperr.NewBadRequest("invalid tenantID")
	}
	st, err := multi.ParseTenantState(state)
	if err != nil {
		return nil, apperr.NewBadRequest("invalid tenantState")
	}
	ddb := app.DDB()
	ctx := c.Request.Context()
	return authx.SetTenantState(ctx, ddb, th, st)
}
