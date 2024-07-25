package server

import (
	"fmt"
	"net/http"

	"github.com/apresdev/landlord/internal/db"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/gin-gonic/gin"
)

func handleRoot(
	c *gin.Context,
	ddb db.AWSDynamoDB,
	awsCfg aws.Config,
	validator authx.TokenValidator) error {

	username, _ := c.Cookie(LandlordUsernameCookie)
	if username == "" {
		c.Redirect(http.StatusFound, "/login")
		return nil
	}

	var rawTokens authx.RawTokens
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
	validationURL := authx.ValidationURL(awsCfg.Region, tenant.UserPoolID)
	tks, err := authx.ValidateAllTokens(
		c.Request.Context(), clientID(c), validator, validationURL, rawTokens)
	if err != nil {
		return err
	}
	idTokDesc, err := authx.JWTDesc("ID Token", tks.IdToken)
	if err != nil {
		return err
	}
	accessTokDesc, err := authx.JWTDesc("Access Token", tks.AccessToken)
	if err != nil {
		return err
	}
	render(c, displayJWTPage(idTokDesc, accessTokDesc))
	return nil
}
