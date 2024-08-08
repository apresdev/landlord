package rest

import (
	"errors"
	"net/http"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/apresdev/landlord/internal/db"

	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func handleResetPassword(app App, c *gin.Context) error {
	var req ResetPasswordRequest
	if err := c.BindJSON(&req); err != nil {
		return apperr.NewBadRequest(err.Error())
	}
	err := HandleResetPassword(app, c, req)
	if err != nil {
		return err
	}
	c.JSON(http.StatusNoContent, nil)
	return nil
}

func HandleResetPassword(app App, c *gin.Context, req ResetPasswordRequest) error {
	username := req.Username
	if username == "" {
		return apperr.NewBadRequest("username is required")
	}
	code := req.ConfirmationCode
	if code == "" {
		return apperr.NewBadRequest("confirmation-code is required")
	}
	newPassword := req.NewPassword
	if newPassword == "" {
		return apperr.NewBadRequest("new-password is required")
	}
	ctx := c.Request.Context()
	ddb := app.DDB()
	cogIdp := app.CogIdp()
	_, tenant, err := db.GetUserAndTenant(ctx, ddb, username)
	if err != nil {
		zap.L().Warn("in reset password, no user/tenant found",
			zap.Error(err),
			zap.String("username", username))
		return err
	}

	upc, err := authx.GetUserPoolClient(ctx, cogIdp, tenant.UserPoolID)
	if err != nil {
		return err
	}
	in := &cognitoidentityprovider.ConfirmForgotPasswordInput{
		ClientId:         upc.ClientId,
		Username:         aws.String(username),
		ConfirmationCode: aws.String(code),
		Password:         aws.String(newPassword),
	}
	if upc.ClientSecret != nil {
		secretHash, err := authx.CreateSecretHash(
			username, aws.ToString(upc.ClientId), aws.ToString(upc.ClientSecret))
		if err != nil {
			return err
		}
		in.SecretHash = aws.String(secretHash)
	}
	_, err = cogIdp.ConfirmForgotPassword(ctx, in)
	if err != nil {
		zap.L().Warn("Failed to confirm forgot password",
			zap.String("username", username),
			zap.Error(err))
		var codeMismatch *types.CodeMismatchException
		if errors.As(err, &codeMismatch) {
			zap.L().Warn("Code mismatch", zap.String("username", username), zap.Error(err))
			return apperr.NewUnauthorized("not authorized")
		}
		return err
	}
	return nil
}
