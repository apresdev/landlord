package rest

import (
	"net/http"
	"slices"

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

func handleImpersonateRole(a App, c *gin.Context) error {
	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		return err
	}

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

	targetTenantHandle, err := multi.ParseTenantHandle(req.Tenant)
	if err != nil {
		return apperr.NewBadRequest(err.Error())
	}
	targetTenant, err := db.ResolveTenant(
		c.Request.Context(), a.DBCtx(), targetTenantHandle)
	if err != nil {
		return apperr.NewBadRequest(err.Error())
	}

	resp, err := HandleImpersonateRole(a, c, sub, targetTenant, req.Role, req.RefreshToken)
	if err != nil {
		return err
	}

	c.JSON(http.StatusOK, resp)

	return nil
}

func HandleImpersonateRole(
	a App,
	c *gin.Context,
	sub *perms.Subject,
	targetTenant *multi.Tenant,
	targetRole string,
	refreshToken string) (*ImpersonateResponse, error) {

	if idx := slices.Index(targetTenant.Roles, targetRole); idx < 0 {
		zap.L().Info("tenant does not have role",
			zap.String("clientID", string(sub.Cid)),
			zap.String("role", targetRole))
		return nil, apperr.NewUnauthorized("not authorized")
	}

	resource := perms.NewResource(
		perms.ResourceKindImpersonation, targetRole, targetTenant, nil)
	effect, err := perms.Eval(c, a, sub, perms.ActionWrite, resource)
	if err != nil {
		zap.L().Error("could not evaluate permissions", zap.Error(err))
		return nil, apperr.NewUnauthorized("not authorized")
	}
	if effect.IsDenied() {
		zap.L().Info("denied impersonate role",
			zap.String("subject", sub.String()),
			zap.String("resource", resource.String()))
		return nil, apperr.NewUnauthorized("not authorized")
	}

	targetGroupName := targetTenant.GroupName(targetRole)
	err = authx.ImpersonateRole(
		c.Request.Context(), a.CogIdp(), sub.Username, targetTenant, targetGroupName)
	if err != nil {
		zap.L().Error("could not set impersonate on user",
			zap.String("clientID", string(sub.Cid)),
			zap.String("userPoolID", targetTenant.UserPoolID),
			zap.String("username", sub.Username),
			zap.String("targetTenantID", targetTenant.ID.String()),
			zap.String("targetGroupName", targetGroupName),
			zap.Error(err))
		return nil, apperr.NewUnauthorized("not authorized")
	}
	upc, err := authx.GetUserPoolClient(
		c.Request.Context(), a.CogIdp(), targetTenant.UserPoolID, a.Settings().UserPoolClient())
	if err != nil {
		zap.L().Warn("Could not resolve user pool client",
			zap.String("userPoolID", targetTenant.UserPoolID),
			zap.Error(err))
		return nil, err
	}
	subClaim, err := jwx.GetClaimString(sub.AccessToken, authx.SubClaim)
	if err != nil {
		zap.L().Error("could not get sub claim from token",
			zap.String("clientID", string(sub.Cid)),
			zap.Error(err))
		return nil, apperr.NewUnauthorized("not authorized")
	}
	params := make(map[string]string)
	params["REFRESH_TOKEN"] = refreshToken
	if upc.ClientSecret != nil {
		secretHash, err := authx.CreateSecretHash(
			subClaim, aws.ToString(upc.ClientId), aws.ToString(upc.ClientSecret))
		if err != nil {
			return nil, err
		}
		params["SECRET_HASH"] = secretHash
	}
	// https://docs.aws.amazon.com/cognito/latest/developerguide/signing-up-users-in-your-app.html#cognito-user-pools-computing-secret-hash
	o, err := a.CogIdp().InitiateAuth(
		c.Request.Context(), &cognitoidentityprovider.InitiateAuthInput{
			AuthFlow:       types.AuthFlowTypeRefreshToken,
			ClientId:       upc.ClientId,
			AuthParameters: params,
		})
	if err != nil {
		zap.L().Error("could not initiate auth",
			zap.String("clientID", string(sub.Cid)),
			zap.Error(err))
		return nil, apperr.NewUnauthorized("not authorized")
	}

	var resp ImpersonateResponse
	resp.IdToken = aws.ToString(o.AuthenticationResult.IdToken)
	resp.AccessToken = aws.ToString(o.AuthenticationResult.AccessToken)
	return &resp, nil
}
