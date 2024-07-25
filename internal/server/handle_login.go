package server

import (
	"net/http"

	"github.com/apresdev/landlord/internal/rest"

	"github.com/apresdev/landlord/internal/appenv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func handleLoginSubmit(a *app, c *gin.Context) error {
	var req rest.LoginRequest
	req.Username = c.PostForm("username")
	req.Password = c.PostForm("password")
	resp, err := rest.HandleLogin(a, c, req)
	if err != nil {
		return err
	}
	switch resp.Status {
	case rest.LoginStatusSuccess:
		setCookie(c, appenv.AppURL(), LandlordUsernameCookie, req.Username)
		setCookie(c, appenv.AppURL(), LandlordAccessTokenCookie, resp.AccessToken)
		setCookie(c, appenv.AppURL(), LandlordIDTokenCookie, resp.IdToken)
		setCookie(c, appenv.AppURL(), LandlordRefreshTokenCookie, resp.RefreshToken)
		c.Redirect(http.StatusFound, "/")
	case rest.LoginStatusNewPasswordRequired:
		setCookie(c, appenv.AppURL(), LandlordUsernameCookie, req.Username)
		setCookie(c, appenv.AppURL(), LandlordChallengeSessionCookie, resp.Session)
		c.Redirect(http.StatusFound, "/new-password")
	case rest.LoginStatusPasswordResetRequired:
		setCookie(c, appenv.AppURL(), LandlordUsernameCookie, req.Username)
		setCookie(c, appenv.AppURL(), LandlordChallengeSessionCookie, resp.Session)
		c.Redirect(http.StatusFound, "/reset-password")
	default:
		zap.L().Error("unexpected login status value", zap.String("status", resp.Status))
		c.String(http.StatusInternalServerError, "internal error")
	}
	return nil
}
