package rest

import (
	"net/http"

	"github.com/apresdev/landlord/internal/db"

	"github.com/apresdev/landlord/internal/multi"

	"github.com/apresdev/landlord/internal/apperr"
	"github.com/gin-gonic/gin"
)

func handleSetTenantDescription(app App, c *gin.Context) error {
	id := c.Param("tenantID")
	if id == "" {
		return apperr.NewBadRequest("no tenantID provided")
	}
	var req TenantSetDescriptionRequest
	if err := c.BindJSON(&req); err != nil {
		return apperr.NewBadRequest("invalid request")
	}
	if _, err := HandleSetTenantDescription(app, c, id, req); err != nil {
		return err
	}
	c.JSON(http.StatusNoContent, nil)
	return nil
}

func HandleSetTenantDescription(
	app App, c *gin.Context, tenantID string, req TenantSetDescriptionRequest) (*multi.Tenant, error) {

	if err := multi.ValidateTenantDescription(req.Description); err != nil {
		return nil, err
	}

	th, err := multi.ParseTenantHandle(tenantID)
	if err != nil {
		return nil, apperr.NewBadRequest("invalid tenantID")
	}
	ddb := app.DDB()
	ctx := c.Request.Context()
	ten, err := db.ResolveTenant(ctx, ddb, th)
	if err != nil {
		return nil, err
	}
	ten.Description = req.Description
	err = db.StoreTenant(ctx, ddb, ten)
	if err != nil {
		return nil, err
	}
	return ten, nil
}
