package server

import (
	"net/http"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/apresdev/landlord/internal/rest"

	"github.com/gin-gonic/gin"
)

func handleCancelInvitation(a *app, c *gin.Context) error {
	tenantID := c.Param("tenantID")
	if tenantID == "" {
		return apperr.NewBadRequest("no tenant ID provided")
	}
	username := c.Param("username")
	if username == "" {
		return apperr.NewBadRequest("no user ID provided")
	}
	if err := rest.HandleCancelInvitation(a, c, username); err != nil {
		return err
	}
	c.String(http.StatusOK, "<div></div>")
	return nil
}
