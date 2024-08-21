package server

import (
	"net/http"

	"go.uber.org/zap"

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
	if resp.Status == rest.LoginStatusMFASetupRequired {
		setCookie(c, appenv.AppURL(), LandlordOTPURI, resp.OtpURI)
		setCookie(c, appenv.AppURL(), LandlordAccessTokenCookie, resp.AccessToken)
		c.Redirect(http.StatusFound, "/setup-otp")
		return nil
	}
	if resp.Status != rest.LoginStatusSuccess {
		zap.L().Info("error setting new password",
			zap.String("username", req.Username),
			zap.String("status", string(resp.Status)))
		c.String(http.StatusUnauthorized, "unauthorized")
		return nil
	}
	setCookie(c, appenv.AppURL(), LandlordIDTokenCookie, resp.IdToken)
	setCookie(c, appenv.AppURL(), LandlordAccessTokenCookie, resp.AccessToken)
	setCookie(c, appenv.AppURL(), LandlordRefreshTokenCookie, resp.RefreshToken)
	clearCookie(c, appenv.AppURL(), LandlordChallengeSessionCookie)
	c.Redirect(http.StatusFound, "/")
	return nil
}
