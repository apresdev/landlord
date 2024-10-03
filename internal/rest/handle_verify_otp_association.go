package rest

import (
	"net/http"

	"github.com/apresdev/landlord/internal/multi"

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
	user, ten, err := db.GetUserAndTenant(ctx, app.DDB(), req.Username)
	if err != nil {
		return apperr.NewBadRequest(err.Error())
	}
	err = authx.VerifySoftwareToken(
		ctx, app.CogIdp(), ten.UserPoolID, req.AccessToken, req.Username, req.OtpCode)
	if err != nil {
		return apperr.NewBadRequest(err.Error())
	}
	if user.Status == multi.UserStatusPendingMFA {
		user.Status = multi.UserStatusActive
		err = db.StoreUser(ctx, app.DDB(), user)
		if err != nil {
			return apperr.NewInternal(err.Error())
		}
	}
	return nil
}
