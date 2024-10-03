package rest

import (
	"net/http"

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
	rc, err := requesterContext(c)
	if err != nil {
		return nil, err
	}
	result, err := authx.AssociateSoftwareToken(
		ctx, app.CogIdp(), app.Settings().AppName(), rc.accessToken.Raw, rc.username)
	if err != nil {
		return nil, err
	}
	var resp AssociateOtpResponse
	resp.OtpURI = result.OtpAuthURI
	return &resp, nil
}
