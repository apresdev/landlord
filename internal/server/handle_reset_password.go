package server

import (
	"net/http"

	"github.com/apresdev/landlord/internal/rest"

	"github.com/gin-gonic/gin"
)

func handleResetPassword(a *app, c *gin.Context) error {
	var req rest.ResetPasswordRequest
	req.Username = c.PostForm("username")
	req.ConfirmationCode = c.PostForm("confirmation-code")
	pwdOne := c.PostForm("password-one")
	pwdTwo := c.PostForm("password-two")
	if pwdOne != pwdTwo {
		c.String(http.StatusBadRequest, "passwords do not match")
		return nil
	}
	req.NewPassword = pwdOne
	if err := rest.HandleResetPassword(a, c, req); err != nil {
		return err
	}
	redirectToLogin(c)
	return nil
}
