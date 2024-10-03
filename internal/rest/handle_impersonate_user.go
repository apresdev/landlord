package rest

import (
	"net/http"

	"github.com/apresdev/landlord/internal/jwx"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/apresdev/landlord/internal/db"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func handleImpersonateUser(app App, c *gin.Context) error {
	var req ImpersonateUserRequest
	if err := c.BindJSON(&req); err != nil {
		return apperr.NewBadRequest(err.Error())
	}
	if req.Tenant == "" {
		return apperr.NewBadRequest("tenant is required")
	}
	if req.Username == "" {
		return apperr.NewBadRequest("username is required")
	}
	if req.RefreshToken == "" {
		return apperr.NewBadRequest("refresh token is required")
	}

	rc, err := requesterContext(c)
	if err != nil {
		return err
	}
	if err := isRequestAuthorizedForRoot(app, c, rc); err != nil {
		return err
	}
	ten, err := db.ResolveTenant(c.Request.Context(), app.DDB(), rc.th)
	if err != nil {
		return apperr.NewBadRequest(err.Error())
	}
	err = authx.ImpersonateUser(
		c.Request.Context(), app.CogIdp(), ten.UserPoolID, rc.username, req.Username)
	if err != nil {
		zap.L().Error("could not set impersonate on user",
			zap.String("clientID", string(rc.cid)),
			zap.String("userPoolID", ten.UserPoolID),
			zap.String("username", rc.username),
			zap.String("impersonateUsername", req.Username),
			zap.Error(err))
		return apperr.NewUnauthorized("not authorized")
	}
	upc, err := authx.GetUserPoolClient(
		c.Request.Context(), app.CogIdp(), ten.UserPoolID)
	if err != nil {
		zap.L().Warn("Could not resolve user pool client",
			zap.String("userPoolID", ten.UserPoolID),
			zap.Error(err))
		return err
	}
	sub, err := jwx.GetClaimString(rc.accessToken, authx.SubClaim)
	if err != nil {
		zap.L().Error("could not get sub from token",
			zap.String("clientID", string(rc.cid)),
			zap.Error(err))
		return apperr.NewUnauthorized("not authorized")
	}
	params := make(map[string]string)
	params["REFRESH_TOKEN"] = req.RefreshToken
	if upc.ClientSecret != nil {
		secretHash, err := authx.CreateSecretHash(
			sub, aws.ToString(upc.ClientId), aws.ToString(upc.ClientSecret))
		if err != nil {
			return err
		}
		params["SECRET_HASH"] = secretHash
	}
	// https://docs.aws.amazon.com/cognito/latest/developerguide/signing-up-users-in-your-app.html#cognito-user-pools-computing-secret-hash
	o, err := app.CogIdp().InitiateAuth(
		c.Request.Context(), &cognitoidentityprovider.InitiateAuthInput{
			AuthFlow:       types.AuthFlowTypeRefreshToken,
			ClientId:       upc.ClientId,
			AuthParameters: params,
		})
	if err != nil {
		zap.L().Error("could not initiate auth",
			zap.String("clientID", string(rc.cid)),
			zap.Error(err))
		return apperr.NewUnauthorized("not authorized")
	}

	var resp ImpersonateResponse
	resp.IdToken = aws.ToString(o.AuthenticationResult.IdToken)
	resp.AccessToken = aws.ToString(o.AuthenticationResult.AccessToken)
	c.JSON(http.StatusOK, resp)

	return nil
}
