package webapp

import (
	"net/http"

	"github.com/apresdev/landlord/internal/rest"

	"github.com/gin-gonic/gin"
)

func handleSetupOTPSubmit(a App, c *gin.Context) error {
	var req rest.VerifyOtpAssociationRequest
	req.Username = c.PostForm("username")
	req.OtpCode = c.PostForm("code")
	req.AccessToken, _ = c.Cookie(LandlordAccessTokenCookie)
	if req.AccessToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing access token"})
		return nil
	}
	err := rest.HandleVerifyOTPAssociation(a, c, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil
	}
	// TODO: add refresh token to request and re-fetch tokens here
	redirectToAppRoot(c)
	return nil
}
