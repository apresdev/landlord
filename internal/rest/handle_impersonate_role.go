package rest

import (
	"net/http"
	"slices"

	"github.com/apresdev/landlord/internal/multi"

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

func handleImpersonateRole(app App, c *gin.Context) error {
	var req ImpersonateRoleRequest
	if err := c.BindJSON(&req); err != nil {
		return apperr.NewBadRequest(err.Error())
	}
	if req.Tenant == "" {
		return apperr.NewBadRequest("tenant is required")
	}
	if req.Role == "" {
		return apperr.NewBadRequest("role is required")
	}
	if req.RefreshToken == "" {
		return apperr.NewBadRequest("refresh token is required")
	}

	rc, err := authx.RequesterFromContext(c)
	if err != nil {
		return err
	}
	// TODO: expand this permission check
	if err := isRequestAuthorizedForRoot(app, c, rc); err != nil {
		return err
	}

	targetTenantHandle, err := multi.ParseTenantHandle(req.Tenant)
	if err != nil {
		return apperr.NewBadRequest(err.Error())
	}
	targetTenant, err := db.ResolveTenant(
		c.Request.Context(), app.DDB(), targetTenantHandle)
	if err != nil {
		return apperr.NewBadRequest(err.Error())
	}
	if idx := slices.Index(targetTenant.Roles, req.Role); idx < 0 {
		zap.L().Info("tenant does not have role",
			zap.String("clientID", string(rc.Cid)),
			zap.String("role", req.Role))
		return apperr.NewUnauthorized("not authorized")
	}

	targetGroupName := targetTenant.GroupName(req.Role)
	err = authx.ImpersonateRole(
		c.Request.Context(),
		app.CogIdp(),
		targetTenant.UserPoolID,
		rc.Username,
		targetTenant.ID,
		targetGroupName)
	if err != nil {
		zap.L().Error("could not set impersonate on user",
			zap.String("clientID", string(rc.Cid)),
			zap.String("userPoolID", targetTenant.UserPoolID),
			zap.String("username", rc.Username),
			zap.String("targetTenantID", targetTenant.ID.String()),
			zap.String("targetGroupName", targetGroupName),
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
