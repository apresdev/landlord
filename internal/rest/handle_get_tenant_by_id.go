package rest

import (
	"net/http"

	"github.com/apresdev/landlord/internal/multi"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/apresdev/landlord/internal/db"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/gin-gonic/gin"
)

func handleGetTenantByID(app App, c *gin.Context) error {
	rc, err := requesterContext(c)
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
	ten, err := db.ResolveTenant(c.Request.Context(), app.DDB(), th)
	if err != nil {
		return apperr.NewNotFound(err.Error())
	}
	if !authx.IsAuthorizedForTenant(app.RootTenantID(), ten, rc.roles) {
		return apperr.NewUnauthorized("not authorized")
	}
	var resp TenantResponse
	resp.Tenant = convertTenant(ten)
	c.JSON(http.StatusOK, resp)
	return nil
}
