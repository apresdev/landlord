package server

import (
	"github.com/apresdev/landlord/internal/db"
	"go.uber.org/zap"

	"github.com/apresdev/landlord/internal/jwx"
	"github.com/gin-gonic/gin"
)

func handleAdminProfile(a *app, c *gin.Context) error {
	var rawTokens jwx.RawTokens
	rawTokens.IdToken, _ = c.Cookie(LandlordIDTokenCookie)
	rawTokens.AccessToken, _ = c.Cookie(LandlordAccessTokenCookie)
	rawTokens.RefreshToken, _ = c.Cookie(LandlordRefreshTokenCookie)
	if rawTokens.IdToken == "" || rawTokens.AccessToken == "" || rawTokens.RefreshToken == "" {
		redirectToLogin(c)
		return nil
	}
	username, _ := c.Cookie(LandlordUsernameCookie)
	if username == "" {
		redirectToLogin(c)
		return nil
	}
	_, tenant, err := db.GetUserAndTenant(c.Request.Context(), a.DDB(), username)
	if err != nil {
		zap.L().Warn("error finding tenant for username",
			zap.String("username", username),
			zap.Error(err))
		clearLandlordCookies(c, a.Settings().URL())
		redirectToLogin(c)
		return nil
	}
	validationURL := jwx.ValidationURL(a.Cfg().Region, tenant.UserPoolID)
	tks, err := jwx.ValidateAllTokens(
		c.Request.Context(), clientID(c), a.Validator(), validationURL, rawTokens)
	if err != nil {
		redirectToLogin(c)
		return err
	}

	idTokDesc, err := jwx.Describe("ID Token", tks.IdToken)
	if err != nil {
		return err
	}
	accessTokDesc, err := jwx.Describe("Access Token", tks.AccessToken)
	if err != nil {
		return err
	}
	render(c, displayJWTPage(idTokDesc, accessTokDesc))
	return nil
}
