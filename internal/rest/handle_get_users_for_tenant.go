package rest

import (
	"net/http"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"go.uber.org/zap"

	"github.com/apresdev/landlord/internal/multi"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/apresdev/landlord/internal/db"

	"github.com/gin-gonic/gin"
)

func handleGetUsersForTenant(app App, c *gin.Context) error {
	rc, err := authx.RequesterFromContext(c)
	if err != nil {
		return err
	}
	t := c.Param("tenantID")
	if t == "" {
		return apperr.NewBadRequest("no tenantID")
	}
	th, err := multi.ParseTenantHandle(t)
	if err != nil {
		return apperr.NewBadRequest(err.Error())
	}
	ten, err := db.ResolveTenant(c.Request.Context(), app.DDB(), th)
	if err != nil {
		return apperr.NewNotFound(err.Error())
	}
	if !authx.IsAuthorizedForTenant(app.RootTenantID(), ten, rc.Roles) {
		zap.L().Info("user not unauthorized for tenant",
			zap.String("clientID", string(rc.Cid)),
			zap.String("username", rc.Username),
			zap.String("tenantID", ten.ID.String()))
		return apperr.NewUnauthorized("not authorized")
	}
	users, err := db.GetUsersForTenant(c.Request.Context(), app.DDB(), ten.ID)
	if err != nil {
		return apperr.NewInternal(err.Error())
	}
	var resp TenantUsersResponse
	for _, user := range users {
		u, err := app.CogIdp().AdminGetUser(
			c.Request.Context(),
			&cognitoidentityprovider.AdminGetUserInput{
				UserPoolId: aws.String(ten.UserPoolID),
				Username:   aws.String(user.Username),
			})
		if err != nil {
			zap.L().Error("error getting user",
				zap.String("username", user.Username),
				zap.String("tenantID", ten.ID.String()),
				zap.Error(err))
			return apperr.NewInternal(err.Error())
		}
		resp.Users = append(resp.Users, TenantUsersResponseUsersInner{
			Username:  user.Username,
			Role:      user.Role,
			Status:    string(user.Status),
			CognitoID: aws.ToString(u.Username),
		})
	}
	c.JSON(http.StatusOK, resp)
	return nil
}
