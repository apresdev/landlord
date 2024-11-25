package rest

import (
	"net/http"

	"github.com/apresdev/landlord/internal/perms"

	"github.com/apresdev/landlord/internal/jwx"

	"github.com/apresdev/landlord/internal/apperr"

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
	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		return err
	}

	// The target user and tenant of this API are from the subject
	targetUser, err := sub.ResolveUser(c.Request.Context(), app)
	if err != nil {
		zap.L().Error("could not resolve subject user", zap.Error(err))
		return apperr.NewUnauthorized("not authorized")
	}
	targetTenant, err := sub.ResolveTenant(c.Request.Context(), app)
	if err != nil {
		zap.L().Error("could not resolve subject tenant", zap.Error(err))
		return apperr.NewUnauthorized("not authorized")
	}

	resource := perms.NewResource(
		perms.ResourceKindImpersonation, targetUser.Username, targetTenant, targetUser)
	effect, err := perms.Eval(c, app, sub, perms.ActionDelete, resource)
	if err != nil {
		zap.L().Error("could not evaluate permissions", zap.Error(err))
		return apperr.NewUnauthorized("not authorized")
	}
	if effect.IsDenied() {
		zap.L().Info("denied clearing impersonation",
			zap.String("subject", sub.String()),
			zap.String("resource", resource.String()))
		return apperr.NewUnauthorized("not authorized")
	}

	err = authx.ClearImpersonate(
		c.Request.Context(), app.CogIdp(), targetTenant.UserPoolID, targetUser.Username)
	if err != nil {
		zap.L().Error("could not clear impersonation",
			zap.String("clientID", string(sub.Cid)),
			zap.Error(err))
		return apperr.NewUnauthorized("not authorized")
	}
	upc, err := authx.GetUserPoolClient(
		c.Request.Context(), app.CogIdp(), targetTenant.UserPoolID)
	if err != nil {
		zap.L().Warn("Could not resolve user pool client",
			zap.String("userPoolID", targetTenant.UserPoolID),
			zap.Error(err))
		return err
	}
	subClaim, err := jwx.GetClaimString(sub.AccessToken, authx.SubClaim)
	if err != nil {
		zap.L().Error("could not get sub claim from token",
			zap.String("clientID", string(sub.Cid)),
			zap.Error(err))
		return apperr.NewUnauthorized("not authorized")
	}
	params := make(map[string]string)
	params["REFRESH_TOKEN"] = req.RefreshToken
	if upc.ClientSecret != nil {
		secretHash, err := authx.CreateSecretHash(
			subClaim, aws.ToString(upc.ClientId), aws.ToString(upc.ClientSecret))
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
			zap.String("clientID", string(sub.Cid)),
			zap.Error(err))
		return apperr.NewUnauthorized("not authorized")
	}

	var resp ImpersonateResponse
	resp.IdToken = aws.ToString(o.AuthenticationResult.IdToken)
	resp.AccessToken = aws.ToString(o.AuthenticationResult.AccessToken)
	c.JSON(http.StatusOK, resp)

	return nil
}
