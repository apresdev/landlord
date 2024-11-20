package rest

import (
	"net/http"

	"github.com/apresdev/landlord/internal/perms"

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

	p, err := perms.PrincipalFromContext(c)
	if err != nil {
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
	targetUser, err := db.GetUserByName(c.Request.Context(), app.DDB(), req.Username)
	if err != nil {
		return apperr.NewBadRequest(err.Error())
	}
	if targetUser.TenantID != targetTenant.ID {
		zap.L().Info("user does not belong to tenant",
			zap.String("clientID", string(p.Cid)),
			zap.String("username", req.Username),
			zap.String("tenantID", targetTenant.ID.String()))
		return apperr.NewUnauthorized("not authorized")
	}
	targetGroupName := targetTenant.GroupName(targetUser.Role)

	resource := perms.NewResource(
		perms.ResourceKindImpersonation, req.Username, targetTenant, targetUser)
	effect, err := perms.Eval(c, app, p, perms.OperationWrite, resource)
	if err != nil {
		zap.L().Error("could not evaluate permissions", zap.Error(err))
		return apperr.NewUnauthorized("not authorized")
	}
	if effect.IsDenied() {
		zap.L().Info("denied impersonate role",
			zap.String("principal", p.String()),
			zap.String("resource", resource.String()))
		return apperr.NewUnauthorized("not authorized")
	}

	err = authx.ImpersonateUser(
		c.Request.Context(),
		app.CogIdp(),
		targetTenant.UserPoolID,
		p.Username,
		targetTenant.ID,
		targetGroupName,
		targetUser.Username,
		targetUser.ID)
	if err != nil {
		zap.L().Error("could not set impersonate on user",
			zap.String("clientID", string(p.Cid)),
			zap.String("userPoolID", targetTenant.UserPoolID),
			zap.String("username", p.Username),
			zap.String("targetTenant", targetTenant.ID.String()),
			zap.String("targetGroupName", targetGroupName),
			zap.String("targetUsername", targetUser.Username),
			zap.String("targetUserID", targetUser.ID.String()),
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
	sub, err := jwx.GetClaimString(p.AccessToken, authx.SubClaim)
	if err != nil {
		zap.L().Error("could not get sub from token",
			zap.String("clientID", string(p.Cid)),
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
			zap.String("clientID", string(p.Cid)),
			zap.Error(err))
		return apperr.NewUnauthorized("not authorized")
	}

	var resp ImpersonateResponse
	resp.IdToken = aws.ToString(o.AuthenticationResult.IdToken)
	resp.AccessToken = aws.ToString(o.AuthenticationResult.AccessToken)
	c.JSON(http.StatusOK, resp)

	return nil
}
