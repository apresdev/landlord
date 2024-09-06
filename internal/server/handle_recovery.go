package server

import (
	"net/http"

	"github.com/apresdev/landlord/internal/rest"

	"github.com/gin-gonic/gin"
)

func handleRecoverySubmit(a *app, c *gin.Context) error {
	var req rest.RecoverPasswordRequest
	req.Username = c.PostForm("username")
	resp, err := rest.HandleRecoverPassword(a, c, req)
	if err != nil {
		return err
	}
	// TODO: put the details from the response into front-end
	_ = resp
	setCookie(c, a.Settings().URL(), LandlordUsernameCookie, req.Username)
	c.Redirect(http.StatusFound, "/reset-password")
	return err
}
