package server

import (
	"net/http"

	"github.com/apresdev/landlord/internal/rest"

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
		setCookie(c, a.Settings().URL(), LandlordUsernameCookie, req.Username)
		setCookie(c, a.Settings().URL(), LandlordAccessTokenCookie, resp.AccessToken)
		setCookie(c, a.Settings().URL(), LandlordIDTokenCookie, resp.IdToken)
		setCookie(c, a.Settings().URL(), LandlordRefreshTokenCookie, resp.RefreshToken)
		redirectToAppRoot(c)
	case rest.LoginStatusNewPasswordRequired:
		setCookie(c, a.Settings().URL(), LandlordUsernameCookie, req.Username)
		setCookie(c, a.Settings().URL(), LandlordChallengeSessionCookie, resp.Session)
		redirectTo(c, "new-password")
	case rest.LoginStatusPasswordResetRequired:
		setCookie(c, a.Settings().URL(), LandlordUsernameCookie, req.Username)
		setCookie(c, a.Settings().URL(), LandlordChallengeSessionCookie, resp.Session)
		redirectTo(c, "reset-password")
	case rest.LoginStatusMFASetupRequired:
		setCookie(c, a.Settings().URL(), LandlordUsernameCookie, req.Username)
		setCookie(c, a.Settings().URL(), LandlordAccessTokenCookie, resp.AccessToken)
		setCookie(c, a.Settings().URL(), LandlordOTPURI, resp.OtpURI)
		redirectTo(c, "setup-otp")
	case rest.LoginStatusMFACodeRequired:
		setCookie(c, a.Settings().URL(), LandlordUsernameCookie, req.Username)
		setCookie(c, a.Settings().URL(), LandlordChallengeSessionCookie, resp.Session)
		redirectTo(c, "otp")
	default:
		zap.L().Error("unexpected login status value",
			zap.String("status", string(resp.Status)))
		c.String(http.StatusInternalServerError, "internal error")
	}
	return nil
}
