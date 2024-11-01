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

func handleClearImpersonate(app App, c *gin.Context) error {
	var req ClearImpersonateRequest
	if err := c.BindJSON(&req); err != nil {
		return apperr.NewBadRequest(err.Error())
	}
	if req.RefreshToken == "" {
		return apperr.NewBadRequest("refresh token is required")
	}
	rc, err := authx.RequesterFromContext(c)
	if err != nil {
		return err
	}
	rootTenant, err := db.GetRootTenant(
		c.Request.Context(), app.DDB(), app.RootTenantID())
	if err != nil {
		zap.L().Error("error fetching root tenant", zap.Error(err))
		return apperr.NewUnauthorized("not authorized")
	}
	if !authx.IsAuthorizedForTenant(app.RootTenantID(), rootTenant, rc.Roles) {
		zap.L().Info("not authorized for root",
			zap.String("clientID", string(rc.Cid)),
			zap.Any("groups", rc.Groups))
		return apperr.NewUnauthorized("not authorized")
	}

	username, err := jwx.GetClaimString(rc.AccessToken, authx.EmailClaim)
	if err != nil {
		zap.L().Error("could not get username from token",
			zap.String("clientID", string(rc.Cid)),
			zap.Error(err))
		return apperr.NewUnauthorized("not authorized")
	}
	err = authx.ClearImpersonate(
		c.Request.Context(), app.CogIdp(), rootTenant.UserPoolID, username)
	if err != nil {
		zap.L().Error("could not clear impersonation",
			zap.String("clientID", string(rc.Cid)),
			zap.Error(err))
		return apperr.NewUnauthorized("not authorized")
	}
	upc, err := authx.GetUserPoolClient(
		c.Request.Context(), app.CogIdp(), rootTenant.UserPoolID)
	if err != nil {
		zap.L().Warn("Could not resolve user pool client",
			zap.String("userPoolID", rootTenant.UserPoolID),
			zap.Error(err))
		return err
	}
	sub, err := jwx.GetClaimString(rc.AccessToken, authx.SubClaim)
	if err != nil {
		zap.L().Error("could not get sub from token",
			zap.String("clientID", string(rc.Cid)),
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
			zap.String("clientID", string(rc.Cid)),
			zap.Error(err))
		return apperr.NewUnauthorized("not authorized")
	}

	var resp ImpersonateResponse
	resp.IdToken = aws.ToString(o.AuthenticationResult.IdToken)
	resp.AccessToken = aws.ToString(o.AuthenticationResult.AccessToken)
	c.JSON(http.StatusOK, resp)

	return nil
}
