package server

import (
	"net/http"

	"github.com/apresdev/landlord/internal/rest"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func handleOTPSubmit(a *app, c *gin.Context) error {
	var req rest.ValidateMfaCodeRequest
	req.Username = c.PostForm("username")
	req.MfaCode = c.PostForm("code")
	req.Session, _ = c.Cookie(LandlordChallengeSessionCookie)
	resp, err := rest.HandleValidateMFACode(a, c, req)
	if err != nil {
		return err
	}
	switch resp.Status {
	case rest.LoginStatusSuccess:
		setCookie(c, a.Settings().URL(), LandlordUsernameCookie, req.Username)
		setCookie(c, a.Settings().URL(), LandlordAccessTokenCookie, resp.AccessToken)
		setCookie(c, a.Settings().URL(), LandlordIDTokenCookie, resp.IdToken)
		setCookie(c, a.Settings().URL(), LandlordRefreshTokenCookie, resp.RefreshToken)
		c.Redirect(http.StatusFound, "/")
	case rest.LoginStatusNewPasswordRequired:
		setCookie(c, a.Settings().URL(), LandlordUsernameCookie, req.Username)
		setCookie(c, a.Settings().URL(), LandlordChallengeSessionCookie, resp.Session)
		c.Redirect(http.StatusFound, "/new-password")
	case rest.LoginStatusPasswordResetRequired:
		setCookie(c, a.Settings().URL(), LandlordUsernameCookie, req.Username)
		setCookie(c, a.Settings().URL(), LandlordChallengeSessionCookie, resp.Session)
		c.Redirect(http.StatusFound, "/reset-password")
	default:
		zap.L().Error("unexpected login status value",
			zap.String("status", string(resp.Status)))
		c.String(http.StatusInternalServerError, "internal error")
	}
	return nil
}
