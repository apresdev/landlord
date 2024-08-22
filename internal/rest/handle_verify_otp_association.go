package rest

import (
	"net/http"

	"github.com/apresdev/landlord/internal/apperr"
	"github.com/apresdev/landlord/internal/authx"
	"github.com/apresdev/landlord/internal/db"
	"github.com/gin-gonic/gin"
)

func handleVerifyOTPAssociation(app App, c *gin.Context) error {
	var req VerifyOtpAssociationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return err
	}
	err := HandleVerifyOTPAssociation(app, c, req)
	if err != nil {
		return err
	}
	c.JSON(http.StatusNoContent, nil)
	return nil
}

func HandleVerifyOTPAssociation(
	app App,
	c *gin.Context,
	req VerifyOtpAssociationRequest) error {

	ctx := c.Request.Context()
	_, ten, err := db.GetUserAndTenant(ctx, app.DDB(), req.Username)
	if err != nil {
		return apperr.NewBadRequest(err.Error())
	}
	return authx.VerifySoftwareToken(
		ctx, app.CogIdp(), ten.UserPoolID, req.AccessToken, req.Username, req.OtpCode)
}
