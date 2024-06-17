package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	LandlordUsernameCookie     = "landlord-username"
	LandlordIDTokenCookie      = "landlord-id-token"
	LandlordAccessTokenCookie  = "landlord-access-token"
	LandlordRefreshTokenCookie = "landlord-refresh-token"
	LandlordCookieMaxAge       = 3600
)

func handleRoot(
	c *gin.Context,
	awsCfg aws.Config,
	cogIdp authx.AWSCognitoIdentityProvider,
	validator authx.TokenValidator) {

	username, _ := c.Cookie(LandlordUsernameCookie)
	if username == "" {
		c.Redirect(http.StatusFound, "/username")
		return
	}

	// TODO: ask for username before redirect?

	userPoolName := os.Getenv("LANDLORD_DEFAULT_USER_POOL")
	code := c.Query("code")
	if code == "" {
		var rawTokens authx.RawTokens
		rawTokens.IdToken, _ = c.Cookie(LandlordIDTokenCookie)
		rawTokens.AccessToken, _ = c.Cookie(LandlordAccessTokenCookie)
		rawTokens.RefreshToken, _ = c.Cookie(LandlordRefreshTokenCookie)

		// No code= and no cookies, redirect to authentication URL
		if rawTokens.Empty() {
			redirectToAuthenticationURL(c, cogIdp, userPoolName)
			return
		}

		up, err := authx.GetUserPoolByNameOrID(c.Request.Context(), cogIdp, userPoolName)
		if err != nil {
			c.String(500, "Error getting user pool: %v", err)
			return
		}
		tks, err := authx.ValidateAllTokens(c.Request.Context(), awsCfg.Region, validator, up, rawTokens)
		if err != nil {
			zap.L().Error("Error validating raw tokens", zap.Error(err))
			redirectToAuthenticationURL(c, cogIdp, userPoolName)
			return
		}
		displayTokens(c, tks)
		return
	}

	tks, err := ValidateCode(
		c.Request.Context(), awsCfg.Region, validator, cogIdp, userPoolName, code)
	if err != nil {
		// TODO: clear cookies
		c.String(500, "Error validating code: %v", err)
		return
	}

	c.SetCookie(
		LandlordIDTokenCookie,
		tks.IdToken.Raw,
		LandlordCookieMaxAge,
		"/",
		authx.AppDomainHost(),
		true,
		true)
	c.SetCookie(
		LandlordAccessTokenCookie,
		tks.AccessToken.Raw,
		LandlordCookieMaxAge,
		"/",
		authx.AppDomainHost(),
		true,
		true)
	c.SetCookie(
		LandlordRefreshTokenCookie,
		tks.RefreshToken,
		LandlordCookieMaxAge,
		"/",
		authx.AppDomainHost(),
		true,
		true)

	// Redirect to the root (without the ?code=) with the cookies set
	c.Redirect(http.StatusFound, "/")
}

func redirectToAuthenticationURL(
	c *gin.Context,
	cogIdp authx.AWSCognitoIdentityProvider,
	userPoolName string) {

	u, err := authx.CreateAuthorizeURL(c.Request.Context(), cogIdp, userPoolName, authx.RedirectURI())
	if err != nil {
		c.String(500, "Error creating authorize URL: %v", err)
		return
	}

	c.SetCookie(LandlordIDTokenCookie, "", -1, "/", authx.AppDomainHost(), true, true)
	c.SetCookie(LandlordAccessTokenCookie, "", -1, "/", authx.AppDomainHost(), true, true)
	c.SetCookie(LandlordRefreshTokenCookie, "", -1, "/", authx.AppDomainHost(), true, true)
	c.Redirect(http.StatusFound, u)
}

func displayTokens(c *gin.Context, tks *authx.Tokens) {
	idTokDesc, err := authx.JWTDesc("ID Token", tks.IdToken)
	if err != nil {
		c.String(500, "Error describing JWT: %v", err)
		return
	}
	accessTokDesc, err := authx.JWTDesc("Access Token", tks.AccessToken)
	if err != nil {
		c.String(500, "Error describing JWT: %v", err)
		return
	}
	if err := displayJWTPage(idTokDesc, accessTokDesc).Render(c.Request.Context(), c.Writer); err != nil {
		c.String(500, "Error rendering JWT: %v", err)
		return
	}
}

func buildAuthHeader(clientID, secret string) string {
	auth := fmt.Sprintf("%s:%s", clientID, secret)
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}

type cogErrMsg struct {
	Error string `json:"error"`
}

var InvalidCodeErr = errors.New("invalid code")

func ValidateCode(
	ctx context.Context,
	region string,
	validator authx.TokenValidator,
	idp authx.AWSCognitoIdentityProvider,
	userPoolName string,
	code string) (*authx.Tokens, error) {

	up, err := authx.GetUserPoolByNameOrID(ctx, idp, userPoolName)
	if err != nil {
		return nil, err
	}
	upc, err := authx.GetSingletonUserPoolClient(ctx, idp, up)
	if err != nil {
		return nil, err
	}
	form := url.Values{}
	form.Add("grant_type", "authorization_code")
	form.Add("code", code)
	form.Add("redirect_uri", os.Getenv("LANDLORD_REDIRECT_URI")) //TODO: get from the DefaultRedirectURI in the client
	encodedFormData := form.Encode()
	req, err := http.NewRequest(
		http.MethodPost,
		"https://auth.ryeknot.com/oauth2/token",
		strings.NewReader(encodedFormData))
	if err != nil {
		return nil, err
	}

	authHeader := buildAuthHeader(*upc.ClientId, *upc.ClientSecret)
	req.Header["Content-Type"] = []string{"application/x-www-form-urlencoded"}
	req.Header["Authorization"] = []string{authHeader}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			zap.L().Error("failed to close response body", zap.Error(err))
		}
	}(resp.Body)

	if resp.StatusCode == http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var rawTokens authx.RawTokens
		err = json.Unmarshal(body, &rawTokens)
		if err != nil {
			return nil, err
		}
		tks, err := authx.ValidateAllTokens(ctx, region, validator, up, rawTokens)
		if err != nil {
			return nil, err
		}
		return tks, nil
	} else if resp.StatusCode == http.StatusBadRequest {
		var e cogErrMsg
		err = json.NewDecoder(resp.Body).Decode(&e)
		if err != nil {
			zap.L().Error(
				"could not parse error message from Cognito", zap.Error(err))
			return nil, InvalidCodeErr
		}
		zap.L().Error("invalid code, likely expired",
			zap.String("code", code),
			zap.String("error", e.Error))
		return nil, InvalidCodeErr
	} else if resp.StatusCode == http.StatusUnauthorized {
		zap.L().Error("unauthorized communication to Cognito")
		return nil, InvalidCodeErr
	} else {
		zap.L().Error(
			"unexpected response from Cognito",
			zap.Int("status", resp.StatusCode))
		return nil, InvalidCodeErr
	}
}
