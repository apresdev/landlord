package server

import (
	"net/http"

	"github.com/apresdev/landlord/internal/rest"

	"github.com/apresdev/landlord/internal/appenv"
	"github.com/gin-gonic/gin"
)

func handleNewPasswordSubmit(a *app, c *gin.Context) error {
	var req rest.SetNewPasswordRequest
	req.Username = c.PostForm("username")
	pwdOne := c.PostForm("password-one")
	pwdTwo := c.PostForm("password-two")
	if pwdOne != pwdTwo {
		c.String(http.StatusBadRequest, "passwords do not match")
		return nil
	}
	req.Password = pwdOne
	req.Session, _ = c.Cookie(LandlordChallengeSessionCookie)
	resp, err := rest.HandleSetNewPassword(a, c, req)
	if err != nil {
		return err
	}
	setCookie(c, appenv.AppURL(), LandlordIDTokenCookie, resp.IdToken)
	setCookie(c, appenv.AppURL(), LandlordAccessTokenCookie, resp.AccessToken)
	setCookie(c, appenv.AppURL(), LandlordRefreshTokenCookie, resp.RefreshToken)
	clearCookie(c, appenv.AppURL(), LandlordChallengeSessionCookie)
	c.Redirect(http.StatusFound, "/")
	return nil
}
