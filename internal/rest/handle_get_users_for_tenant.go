package rest

import (
	"net/http"

	"github.com/apresdev/landlord/internal/authx"
	"go.uber.org/zap"

	"github.com/apresdev/landlord/internal/multi"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/apresdev/landlord/internal/db"

	"github.com/gin-gonic/gin"
)

func handleGetUsersForTenant(app App, c *gin.Context) error {
	ur, err := resolveUser(c)
	if err != nil {
		return err
	}
	t := c.Param("tenantID")
	if t == "" {
		return apperr.NewBadRequest("no tenantID")
	}
	th, err := multi.ParseTenantHandle(t)
	if err != nil {
		return apperr.NewBadRequest(err.Error())
	}
	ten, err := db.ResolveTenant(c.Request.Context(), app.DDB(), th)
	if err != nil {
		return apperr.NewNotFound(err.Error())
	}
	if !authx.IsAuthorizedForTenant(ten, ur.Roles) {
		zap.L().Info("user not unauthorized for tenant",
			zap.String("clientID", string(ur.ClientID)),
			zap.String("username", ur.Username),
			zap.String("tenantID", ten.ID.String()))
		return apperr.NewUnauthorized("not authorized")
	}
	users, err := db.GetUsersForTenant(c.Request.Context(), app.DDB(), ten.ID)
	if err != nil {
		return apperr.NewInternal(err.Error())
	}
	var resp TenantUsersResponse
	for _, user := range users {
		resp.Users = append(resp.Users, TenantUsersResponseUsersInner{
			Username: user.Username,
			Role:     user.Role,
			Status:   string(user.Status),
		})
	}
	c.JSON(http.StatusOK, resp)
	return nil
}
