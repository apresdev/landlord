package rest

import (
	"net/http"

	"github.com/apresdev/landlord/internal/apperr"
	"github.com/apresdev/landlord/internal/db"

	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func handleRecoverPassword(app App, c *gin.Context) error {
	var req RecoverPasswordRequest
	if err := c.BindJSON(&req); err != nil {
		return apperr.NewBadRequest(err.Error())
	}
	resp, err := HandleRecoverPassword(app, c, req)
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, resp)
	return nil
}

func HandleRecoverPassword(
	app App, c *gin.Context, req RecoverPasswordRequest) (*RecoverPasswordResponse, error) {

	ctx := c.Request.Context()
	ddb := app.DDB()
	cogIdp := app.CogIdp()
	username := req.Username

	if username == "" {
		return nil, apperr.NewBadRequest("username is required")
	}
	_, tenant, err := db.GetUserAndTenant(ctx, ddb, username)
	if err != nil {
		return nil, err
	}
	upc, err := authx.GetUserPoolClient(
		c.Request.Context(), cogIdp, tenant.UserPoolID)
	if err != nil {
		zap.L().Warn("Could not resolve user pool client",
			zap.String("tenantID", tenant.ID.String()),
			zap.String("userPoolID", tenant.UserPoolID),
			zap.Error(err))
		return nil, err
	}
	in := &cognitoidentityprovider.ForgotPasswordInput{
		ClientId: upc.ClientId,
		Username: aws.String(username),
	}
	if upc.ClientSecret != nil {
		secretHash, err := authx.CreateSecretHash(
			username, aws.ToString(upc.ClientId), aws.ToString(upc.ClientSecret))
		if err != nil {
			return nil, err
		}
		in.SecretHash = aws.String(secretHash)
	}
	r, err := cogIdp.ForgotPassword(ctx, in)
	if err != nil {
		zap.L().Warn("Error sending forgot password", zap.Error(err))
		return nil, err
	}

	// TODO: handle SMS and email separately?
	zap.L().Info("password reset",
		zap.String("username", username),
		zap.String("attr", aws.ToString(r.CodeDeliveryDetails.AttributeName)),
		zap.String("medium", string(r.CodeDeliveryDetails.DeliveryMedium)),
		zap.String("destination", aws.ToString(r.CodeDeliveryDetails.Destination)))

	var resp RecoverPasswordResponse
	resp.DeliveryMedium = string(r.CodeDeliveryDetails.DeliveryMedium)
	resp.Destination = aws.ToString(r.CodeDeliveryDetails.Destination)
	return &resp, nil
}
