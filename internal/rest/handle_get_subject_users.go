package rest

import (
	"net/http"

	"github.com/apresdev/landlord/internal/perms"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/gin-gonic/gin"
)

func handleGetSubjectUsers(app App, c *gin.Context) error {
	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		return err
	}
	ten, err := sub.ResolveTenant(c.Request.Context(), app)
	if err != nil {
		return apperr.NewNotFound(err.Error())
	}
	resp, err := getUsersForTenant(app, c, ten, sub)
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, resp)
	return nil
}
