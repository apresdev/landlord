package server

import (
	"github.com/apresdev/landlord/internal/rest"

	"github.com/gin-gonic/gin"
)

func handleNewTenant(a *app, c *gin.Context) error {
	var req rest.CreateTenantRequest
	req.Name = c.PostForm("name")
	req.Description = c.PostForm("description")
	_, err := rest.HandleCreateTenant(a, c, req)
	if err != nil {
		return err
	}
	redirectTo(c, "admin/tenants")
	return nil
}
