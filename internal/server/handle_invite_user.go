package server

import (
	"net/http"

	"github.com/apresdev/landlord/internal/rest"

	"github.com/gin-gonic/gin"
)

func handleInviteUser(a *app, c *gin.Context) error {
	var req rest.InviteRequest
	req.Username = c.PostForm("username")
	req.Tenant = c.Param("tenantID")
	req.Role = c.PostForm("role")
	req.MfaPolicy = rest.MfaPolicy(c.PostForm("mfaPolicy"))
	if err := rest.HandleInviteToTenant(a, c, req); err != nil {
		return err
	}
	c.Redirect(http.StatusFound, "/admin/tenants/"+req.Tenant)
	return nil
}
