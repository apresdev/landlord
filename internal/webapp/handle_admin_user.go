package webapp

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"

	"github.com/apresdev/landlord/internal/authx"

	"github.com/google/uuid"

	"github.com/apresdev/landlord/internal/apperr"
	"github.com/apresdev/landlord/internal/multi"

	"github.com/apresdev/landlord/internal/db"
	"github.com/gin-gonic/gin"
)

func handleAdminUser(a App, c *gin.Context) error {
	cu, err := resolveCurrentUser(a, c)
	if err != nil {
		clearLandlordCookies(c, a.Settings().URL())
		redirectToLogin(c)
		return nil
	}

	tenantID := c.Param("tenantID")
	th, err := multi.ParseTenantHandle(tenantID)
	if err != nil {
		return apperr.NewBadRequestE(
			fmt.Errorf("error parsing tenant ID: %w", err))
	}
	ten, err := db.ResolveTenant(c.Request.Context(), a.DDB(), th)
	if err != nil && !apperr.IsNotFound(err) {
		return apperr.NewNotFoundE(
			fmt.Errorf("error resolving tenant: %w", err))
	}
	if err != nil {
		return apperr.NewInternalE(
			fmt.Errorf("error getting tenants: %w", err))
	}

	userID := c.Param("userID")
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return apperr.NewBadRequestE(
			fmt.Errorf("error parsing user ID: %w", err))
	}
	u, err := db.GetUserInTenantByID(c.Request.Context(), a.DDB(), ten.ID, userUUID)
	if err != nil {
		return apperr.NewNotFoundE(
			fmt.Errorf("error getting user: %w", err))
	}

	cogU, err := a.CogIdp().AdminGetUser(c.Request.Context(), &cognitoidentityprovider.AdminGetUserInput{
		UserPoolId: aws.String(ten.UserPoolID),
		Username:   aws.String(u.Username),
	})
	if err != nil {
		return apperr.NewInternalE(
			fmt.Errorf("error getting user from Cognito: %w", err))
	}
	_ = cogU

	cogURL := authx.CognitoUserDetailsURL(a.Cfg().Region, ten.UserPoolID, u.ID.String())
	render(c, userSummary(a.Settings().AppName(), cu, u, cogURL, ten))

	return nil
}
