package rest

import (
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/apresdev/landlord/internal/multi"

	"github.com/apresdev/landlord/internal/jwx"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/apresdev/landlord/internal/db"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func clientID(c *gin.Context) jwx.OpaqueClientID {
	return jwx.OpaqueClientID(c.ClientIP())
}

func handleLogin(app App, c *gin.Context) error {
	var req LoginRequest
	if err := c.BindJSON(&req); err != nil {
		return apperr.NewBadRequestE(err)
	}
	resp, err := HandleLogin(app, c, req)
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, resp)
	return nil
}

func HandleLogin(app App, c *gin.Context, req LoginRequest) (*LoginResponse, error) {
	if req.Username == "" {
		return nil, apperr.NewBadRequest("username is required")
	}
	if req.Password == "" {
		return nil, apperr.NewBadRequest("password is required")
	}
	ctx := c.Request.Context()
	user, tenant, err := db.GetUserAndTenant(ctx, app.DDB(), req.Username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		zap.L().Info("No user found for username",
			zap.String("username", req.Username))
		return nil, apperr.NewUnauthorized("invalid username or password")
	}
	if tenant == nil {
		zap.L().Info("No tenants found for username",
			zap.String("username", req.Username))
		return nil, apperr.NewUnauthorized("no tenants found for username")
	}
	upc, err := authx.GetUserPoolClient(
		ctx, app.CogIdp(), tenant.UserPoolID)
	if err != nil {
		zap.L().Warn("Could not resolve user pool client",
			zap.String("userPoolID", tenant.UserPoolID), zap.Error(err))
		return nil, err
	}
	params := make(map[string]string)
	params["USERNAME"] = req.Username
	params["PASSWORD"] = req.Password
	if upc.ClientSecret != nil {
		secretHash, err := authx.CreateSecretHash(
			req.Username, aws.ToString(upc.ClientId), aws.ToString(upc.ClientSecret))
		if err != nil {
			return nil, err
		}
		params["SECRET_HASH"] = secretHash
	}
	o, err := app.CogIdp().InitiateAuth(ctx, &cognitoidentityprovider.InitiateAuthInput{
		AuthFlow:       "USER_PASSWORD_AUTH",
		ClientId:       upc.ClientId,
		AuthParameters: params,
	})
	if err != nil {
		zap.L().Warn("Error initiating auth", zap.Error(err))
		var userNotFound *types.UserNotFoundException
		if errors.As(err, &userNotFound) {
			zap.L().Info("User not found", zap.String("username", req.Username))
			return nil, apperr.NewUnauthorized("Invalid username or password")
		}
		var passwordResetRequired *types.PasswordResetRequiredException
		if errors.As(err, &passwordResetRequired) {
			zap.L().Info("Password reset required", zap.String("username", req.Username))
			var resp LoginResponse
			resp.Status = LoginStatusPasswordResetRequired
			return &resp, nil
		}
		zap.L().Info("Authentication failed",
			zap.String("username", req.Username), zap.Error(err))
		return nil, apperr.NewUnauthorized("Invalid username or password")
	}

	if o.ChallengeName == types.ChallengeNameTypeNewPasswordRequired {
		var resp LoginResponse
		resp.Status = LoginStatusNewPasswordRequired
		resp.Session = aws.ToString(o.Session)
		return &resp, nil
	}

	if o.ChallengeName == types.ChallengeNameTypeSoftwareTokenMfa {
		var resp LoginResponse
		resp.Status = LoginStatusMFACodeRequired
		resp.Session = aws.ToString(o.Session)
		return &resp, nil
	}

	if o.ChallengeName != "" {
		zap.L().Warn("Unexpected challenge name",
			zap.String("challengeName", string(o.ChallengeName)))
		return nil, apperr.NewInternal("unexpected challenge name")
	}

	if user.MFAPolicy == multi.MFAPolicyRequired {
		u, err := app.CogIdp().AdminGetUser(ctx,
			&cognitoidentityprovider.AdminGetUserInput{
				UserPoolId: aws.String(tenant.UserPoolID),
				Username:   aws.String(req.Username)})
		if err != nil {
			return nil, apperr.NewInternalE(fmt.Errorf("error getting user: %w", err))
		}
		idx := slices.Index(u.UserMFASettingList, "SOFTWARE_TOKEN_MFA")
		if idx == -1 {
			result, err := authx.AssociateSoftwareToken(
				ctx,
				app.CogIdp(),
				app.Settings().AppName(),
				aws.ToString(o.AuthenticationResult.AccessToken),
				req.Username)
			if err != nil {
				return nil, apperr.NewInternalE(fmt.Errorf("error associating software token: %w", err))
			}
			var resp LoginResponse
			resp.Status = LoginStatusMFASetupRequired
			resp.OtpURI = result.OtpAuthURI
			resp.AccessToken = aws.ToString(o.AuthenticationResult.AccessToken)
			return &resp, nil
		}
	}

	if err := authx.OnSuccessfulLogin(ctx, app.DDB(), app.CogIdp(), tenant, user); err != nil {
		return nil, err
	}

	cid := clientID(c)
	validationURL := jwx.ValidationURL(app.Cfg().Region, tenant.UserPoolID)
	accessTok, err := app.Validator().ValidateToken(
		ctx, cid, validationURL, aws.ToString(o.AuthenticationResult.AccessToken))
	if err != nil {
		return nil, err
	}
	idTok, err := app.Validator().ValidateToken(
		ctx, cid, validationURL, aws.ToString(o.AuthenticationResult.IdToken))
	if err != nil {
		return nil, err
	}

	var resp LoginResponse
	resp.Status = LoginStatusSuccess
	resp.IdToken = idTok.Raw
	resp.AccessToken = accessTok.Raw
	resp.RefreshToken = aws.ToString(o.AuthenticationResult.RefreshToken)
	return &resp, nil
}
