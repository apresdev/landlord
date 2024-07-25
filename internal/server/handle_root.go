package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/apresdev/landlord/internal/jwx"

	"github.com/apresdev/landlord/internal/db"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/gin-gonic/gin"
)

func handleRoot(
	c *gin.Context,
	ddb db.AWSDynamoDB,
	awsCfg aws.Config,
	validator jwx.TokenValidator) error {

	ua := c.GetHeader("User-Agent")
	if strings.Contains(strings.ToLower(ua), "healthcheck") {
		c.JSON(http.StatusNoContent, nil)
		return nil
	}

	username, _ := c.Cookie(LandlordUsernameCookie)
	if username == "" {
		c.Redirect(http.StatusFound, "/login")
		return nil
	}

	var rawTokens jwx.RawTokens
	rawTokens.IdToken, _ = c.Cookie(LandlordIDTokenCookie)
	rawTokens.AccessToken, _ = c.Cookie(LandlordAccessTokenCookie)
	rawTokens.RefreshToken, _ = c.Cookie(LandlordRefreshTokenCookie)
	if rawTokens.IdToken == "" || rawTokens.AccessToken == "" || rawTokens.RefreshToken == "" {
		c.Redirect(http.StatusFound, "/login")
		return nil
	}

	_, tenant, err := db.GetUserAndTenant(c.Request.Context(), ddb, username)
	if err != nil {
		return fmt.Errorf("error finding tenant for username: %w", err)
	}
	validationURL := jwx.ValidationURL(awsCfg.Region, tenant.UserPoolID)
	tks, err := jwx.ValidateAllTokens(
		c.Request.Context(), clientID(c), validator, validationURL, rawTokens)
	if err != nil {
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
