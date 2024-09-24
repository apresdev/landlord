package webapp

import (
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/apresdev/landlord/internal/jwx"

	"github.com/apresdev/landlord/internal/db"

	"github.com/gin-gonic/gin"
)

func handleRoot(a App, c *gin.Context) error {
	ua := c.GetHeader("User-Agent")
	if strings.Contains(strings.ToLower(ua), "healthcheck") {
		c.JSON(http.StatusNoContent, nil)
		return nil
	}

	username, _ := c.Cookie(LandlordUsernameCookie)
	if username == "" {
		redirectToLogin(c)
		return nil
	}

	var rawTokens jwx.RawTokens
	rawTokens.IdToken, _ = c.Cookie(LandlordIDTokenCookie)
	rawTokens.AccessToken, _ = c.Cookie(LandlordAccessTokenCookie)
	rawTokens.RefreshToken, _ = c.Cookie(LandlordRefreshTokenCookie)
	if rawTokens.IdToken == "" || rawTokens.AccessToken == "" || rawTokens.RefreshToken == "" {
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
	_, err = jwx.ValidateAllTokens(
		c.Request.Context(), clientID(c), a.Validator(), validationURL, rawTokens)
	if err != nil {
		redirectToLogin(c)
		return err
	}

	// TODO: make this redirect conditional on the user we just validated
	redirectTo(c, "admin/dashboard")

	return nil
}
