package rest

import (
	"net/http"

	"github.com/apresdev/landlord/internal/perms"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/gin-gonic/gin"
)

func handleAssociateOTP(app App, c *gin.Context) error {
	resp, err := HandleAssociateOTP(app, c)
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, resp)
	return nil
}

func HandleAssociateOTP(app App, c *gin.Context) (*AssociateOtpResponse, error) {
	ctx := c.Request.Context()
	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		return nil, err
	}
	result, err := authx.AssociateSoftwareToken(
		ctx, app.CogIdp(), app.Settings().AppName(), sub.AccessToken.Raw, sub.Username)
	if err != nil {
		return nil, err
	}
	var resp AssociateOtpResponse
	resp.OtpURI = result.OtpAuthURI
	return &resp, nil
}
