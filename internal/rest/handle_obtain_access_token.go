package rest

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/apresdev/landlord/internal/perms"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/apresdev/landlord/internal/db"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type obtainAccessTokenForm struct {
	GrantType    string `json:"grant_type"`
	Code         string `json:"code"`
	ClientID     string `json:"client_id"`
	RedirectURI  string `json:"redirect_uri"`
	CodeVerifier string `json:"code_verifier"`
}

// HandleObtainAccessToken handles an oauth2 request to obtain an access token. This
// is not a traditional JSON POST, but a form POST. Part of the oauth2 authorization
// flow.
func HandleObtainAccessToken(app App, c *gin.Context) (*ObtainTokenResponse, error) {
	if err := c.Request.ParseForm(); err != nil {
		zap.L().Error("Error parsing form", zap.Error(err))
		return nil, apperr.NewBadRequest("bad request")
	}

	var req obtainAccessTokenForm
	req.GrantType = c.PostForm("grant_type")
	req.Code = c.PostForm("code")
	req.ClientID = c.PostForm("client_id")
	req.RedirectURI = c.PostForm("redirect_uri")
	req.CodeVerifier = c.PostForm("code_verifier")

	cid := perms.ClientID(c)
	if req.GrantType == "" {
		zap.L().Info("No grant type in request", zap.String("cid", string(cid)))
		return nil, apperr.NewBadRequest("missing grant type in request")
	}
	if req.Code == "" {
		zap.L().Info("No code in request", zap.String("cid", string(cid)))
		return nil, apperr.NewBadRequest("missing code in request")
	}
	if req.ClientID == "" {
		zap.L().Info("No client ID in request", zap.String("cid", string(cid)))
		return nil, apperr.NewBadRequest("missing client ID in request")
	}
	if req.RedirectURI == "" {
		zap.L().Info("No redirect URI in request", zap.String("cid", string(cid)))
		return nil, apperr.NewBadRequest("missing redirect URI in request")
	}
	if req.CodeVerifier == "" {
		zap.L().Info("No code verifier in request", zap.String("cid", string(cid)))
		return nil, apperr.NewBadRequest("missing code verifier in request")
	}
	ctx := c.Request.Context()
	rootTenant, err := db.GetRootTenant(ctx, app.DDB(), app.RootTenantID())
	if err != nil {
		zap.L().Error("Error getting root tenant", zap.Error(err))
		return nil, apperr.NewUnauthorized("unauthorized")
	}
	up, err := authx.GetUserPoolByID(ctx, app.CogIdp(), rootTenant.UserPoolID)
	if err != nil {
		zap.L().Warn("Could not resolve user pool",
			zap.String("userPoolID", rootTenant.UserPoolID), zap.Error(err))
		return nil, apperr.NewUnauthorized("unauthorized")
	}
	cognitoOauth2TokenEndpoint, err := authx.CognitoOauth2TokenEndpoint(up)
	if err != nil {
		zap.L().Error("Error getting cognito oauth2 token endpoint", zap.Error(err))
		return nil, apperr.NewUnauthorized("unauthorized")
	}
	upc, err := authx.GetUserPoolClient(
		ctx, app.CogIdp(), rootTenant.UserPoolID)
	if err != nil {
		zap.L().Warn("Could not resolve user pool client",
			zap.String("userPoolID", rootTenant.UserPoolID), zap.Error(err))
		return nil, apperr.NewUnauthorized("unauthorized")
	}
	params := url.Values{}
	params.Set("grant_type", req.GrantType)
	params.Set("code", req.Code)
	params.Set("client_id", req.ClientID)
	params.Set("redirect_uri", req.RedirectURI)
	params.Set("code_verifier", req.CodeVerifier)
	if upc.ClientSecret != nil {
		params.Set("client_secret", aws.ToString(upc.ClientSecret))
	}
	upstreamURL := cognitoOauth2TokenEndpoint
	upstreamReq, err := http.NewRequest("POST", upstreamURL, strings.NewReader(params.Encode()))
	if err != nil {
		zap.L().Error("Error creating request", zap.Error(err))
		return nil, apperr.NewUnauthorized("unauthorized")
	}
	upstreamReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{}
	upstreamResp, err := client.Do(upstreamReq)
	if err != nil {
		zap.L().Error("Error sending request", zap.Error(err))
		return nil, apperr.NewUnauthorized("unauthorized")
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			zap.L().Error("Error closing response body", zap.Error(err))
		}
	}(upstreamResp.Body)
	responseBodyBytes, err := io.ReadAll(upstreamResp.Body)
	if err != nil {
		zap.L().Error("Error reading response body", zap.Error(err))
		return nil, apperr.NewUnauthorized("unauthorized")
	}
	var resp ObtainTokenResponse
	err = json.Unmarshal(responseBodyBytes, &resp)
	if err != nil {
		zap.L().Error("Error unmarshalling response", zap.Error(err))
		return nil, apperr.NewUnauthorized("unauthorized")
	}
	if resp.AccessToken == "" {
		zap.L().Error("No access token in response")
		return nil, apperr.NewUnauthorized("unauthorized")
	}
	if resp.IdToken == "" {
		zap.L().Error("No ID token in response")
		return nil, apperr.NewUnauthorized("unauthorized")
	}
	if resp.RefreshToken == "" {
		zap.L().Error("No refresh token in response")
		return nil, apperr.NewUnauthorized("unauthorized")
	}
	if resp.TokenType == "" {
		zap.L().Error("No token type in response")
		return nil, apperr.NewUnauthorized("unauthorized")
	}
	if resp.ExpiresIn == 0 {
		zap.L().Error("No expires in in response")
		return nil, apperr.NewUnauthorized("unauthorized")
	}
	return &resp, nil
}
