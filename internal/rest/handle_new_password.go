package rest

import (
	"fmt"
	"net/http"
	"slices"

	"github.com/apresdev/landlord/internal/multi"

	"github.com/apresdev/landlord/internal/apperr"
	"github.com/apresdev/landlord/internal/db"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func handleSetNewPassword(app App, c *gin.Context) error {
	var req SetNewPasswordRequest
	if err := c.BindJSON(&req); err != nil {
		return apperr.NewBadRequest(err.Error())
	}
	resp, err := HandleSetNewPassword(app, c, req)
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, resp)
	return nil
}

func HandleSetNewPassword(
	app App, c *gin.Context, req SetNewPasswordRequest) (
	*SetNewPasswordResponse, error) {

	ddb := app.DDB()
	cogIdp := app.CogIdp()
	ctx := c.Request.Context()
	user, tenant, err := db.GetUserAndTenant(ctx, ddb, req.Username)
	if err != nil {
		return nil, fmt.Errorf("error finding tenant for username: %w", err)
	}
	upc, err := authx.GetUserPoolClient(ctx, cogIdp, tenant.UserPoolID)
	if err != nil {
		return nil, err
	}
	params := make(map[string]string)
	params["USERNAME"] = req.Username
	params["NEW_PASSWORD"] = req.Password
	if upc.ClientSecret != nil {
		secretHash, err := authx.CreateSecretHash(
			req.Username, aws.ToString(upc.ClientId), aws.ToString(upc.ClientSecret))
		if err != nil {
			return nil, err
		}
		params["SECRET_HASH"] = secretHash
	}
	o, err := cogIdp.RespondToAuthChallenge(ctx,
		&cognitoidentityprovider.RespondToAuthChallengeInput{
			ChallengeName:      types.ChallengeNameTypeNewPasswordRequired,
			ClientId:           upc.ClientId,
			ChallengeResponses: params,
			Session:            aws.String(req.Session)})
	if err != nil {
		return nil, err
	}
	_, err = cogIdp.AdminConfirmSignUp(ctx,
		&cognitoidentityprovider.AdminConfirmSignUpInput{
			UserPoolId: aws.String(tenant.UserPoolID),
			Username:   aws.String(req.Username)})
	if err != nil {
		zap.L().Warn("error confirming user", zap.Error(err))
	}
	// It is safe to confirm the email address here as the temporary password
	// was sent to the email address in question and secured by Cognito's
	// one-time "session" value.
	_, err = cogIdp.AdminUpdateUserAttributes(ctx,
		&cognitoidentityprovider.AdminUpdateUserAttributesInput{
			UserPoolId: aws.String(tenant.UserPoolID),
			Username:   aws.String(req.Username),
			UserAttributes: []types.AttributeType{
				{Name: aws.String("email_verified"), Value: aws.String("true")},
			},
		})
	if err != nil {
		zap.L().Warn("error updating email_verified user attribute", zap.Error(err))
	}

	if user.MFAPolicy == multi.MFAPolicyRequired {
		u, err := cogIdp.AdminGetUser(ctx,
			&cognitoidentityprovider.AdminGetUserInput{
				UserPoolId: aws.String(tenant.UserPoolID),
				Username:   aws.String(req.Username)})
		if err != nil {
			return nil, apperr.NewInternalE(fmt.Errorf("error getting user: %w", err))
		}
		idx := slices.Index(u.UserMFASettingList, "SOFTWARE_TOKEN_MFA")
		if idx == -1 {
			result, err := authx.AssociateSoftwareToken(
				ctx, cogIdp, aws.ToString(o.AuthenticationResult.AccessToken), req.Username)
			if err != nil {
				return nil, apperr.NewInternalE(fmt.Errorf("error associating software token: %w", err))
			}
			var resp SetNewPasswordResponse
			resp.Status = LoginStatusMFASetupRequired
			resp.OtpURI = result.OtpAuthURI
			resp.AccessToken = aws.ToString(o.AuthenticationResult.AccessToken)
			return &resp, nil
		}
	}

	var resp SetNewPasswordResponse
	resp.Status = LoginStatusSuccess
	resp.IdToken = aws.ToString(o.AuthenticationResult.IdToken)
	resp.AccessToken = aws.ToString(o.AuthenticationResult.AccessToken)
	resp.RefreshToken = aws.ToString(o.AuthenticationResult.RefreshToken)
	return &resp, nil
}
