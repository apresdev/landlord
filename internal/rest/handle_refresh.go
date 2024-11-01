package rest

import (
	"net/http"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/apresdev/landlord/internal/db"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func handleRefresh(app App, c *gin.Context) error {
	var req RefreshRequest
	if err := c.BindJSON(&req); err != nil {
		return apperr.NewBadRequestE(err)
	}
	resp, err := HandleRefresh(app, c, req)
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, resp)
	return nil
}

func HandleRefresh(app App, c *gin.Context, req RefreshRequest) (*LoginResponse, error) {
	cid := authx.ClientID(c)
	if req.RefreshToken == "" {
		zap.L().Info("No refresh token in request", zap.String("cid", string(cid)))
		return nil, apperr.NewBadRequest("missing refreshToken in request")
	}
	if req.Username == "" {
		zap.L().Info("No username in request", zap.String("cid", string(cid)))
		return nil, apperr.NewBadRequest("missing username in request")
	}
	ctx := c.Request.Context()
	_, tenant, err := db.GetUserAndTenant(ctx, app.DDB(), req.Username)
	if err != nil {
		return nil, apperr.NewUnauthorized("unauthorized")
	}
	upc, err := authx.GetUserPoolClient(
		ctx, app.CogIdp(), tenant.UserPoolID)
	if err != nil {
		zap.L().Warn("Could not resolve user pool client",
			zap.String("userPoolID", tenant.UserPoolID), zap.Error(err))
		return nil, apperr.NewUnauthorized("unauthorized")
	}

	cogUser, err := app.CogIdp().AdminGetUser(
		ctx, &cognitoidentityprovider.AdminGetUserInput{
			UserPoolId: aws.String(tenant.UserPoolID),
			Username:   aws.String(req.Username),
		})
	if err != nil {
		zap.L().Warn("Error getting user",
			zap.Error(err), zap.String("cid", string(cid)))
		return nil, apperr.NewUnauthorized("unauthorized")
	}
	sub := ""
	for _, userAttr := range cogUser.UserAttributes {
		if *userAttr.Name == "sub" {
			sub = *userAttr.Value
			break
		}
	}
	if sub == "" {
		zap.L().Warn("No sub found in user attributes",
			zap.String("cid", string(cid)))
		return nil, apperr.NewUnauthorized("unauthorized")
	}

	params := make(map[string]string)
	params["REFRESH_TOKEN"] = req.RefreshToken
	if upc.ClientSecret != nil {
		secretHash, err := authx.CreateSecretHash(
			sub, aws.ToString(upc.ClientId), aws.ToString(upc.ClientSecret))
		if err != nil {
			zap.L().Warn("Error creating secret hash",
				zap.Error(err), zap.String("cid", string(cid)))
			return nil, apperr.NewUnauthorized("unauthorized")
		}
		params["SECRET_HASH"] = secretHash
	}
	o, err := app.CogIdp().InitiateAuth(
		c.Request.Context(), &cognitoidentityprovider.InitiateAuthInput{
			AuthFlow:       types.AuthFlowTypeRefreshToken,
			ClientId:       upc.ClientId,
			AuthParameters: params,
		})
	if err != nil {
		zap.L().Warn("Error initiating auth",
			zap.Error(err), zap.String("cid", string(cid)))
		return nil, apperr.NewUnauthorized("unauthorized")
	}

	resp := LoginResponse{}
	resp.Status = LoginStatusSuccess
	resp.IdToken = aws.ToString(o.AuthenticationResult.IdToken)
	resp.AccessToken = aws.ToString(o.AuthenticationResult.AccessToken)
	return &resp, nil
}
