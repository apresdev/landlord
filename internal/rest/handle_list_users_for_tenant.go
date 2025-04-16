package rest

import (
	"net/http"

	"github.com/apresdev/landlord/internal/perms"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"go.uber.org/zap"

	"github.com/apresdev/landlord/internal/multi"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/apresdev/landlord/internal/db"

	"github.com/gin-gonic/gin"
)

func getUsersForTenant(
	a App,
	c *gin.Context,
	ten *multi.Tenant,
	sub *perms.Subject) (*TenantUsersResponse, error) {

	if !perms.IsAuthorizedToReadTenant(a.RootTenantID(), ten, sub.Role) {
		zap.L().Info("user not unauthorized for tenant",
			zap.String("clientID", string(sub.Cid)),
			zap.String("username", sub.Username),
			zap.String("tenantID", ten.ID.String()))
		return nil, apperr.NewUnauthorized("not authorized")
	}
	users, err := db.GetUsersForTenant(c.Request.Context(), a.DBCtx(), ten.ID)
	if err != nil {
		return nil, apperr.NewInternal(err.Error())
	}
	resp := TenantUsersResponse{TenantID: ten.ID.String()}
	for _, user := range users {
		u, err := a.CogIdp().AdminGetUser(
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
			return nil, apperr.NewInternal(err.Error())
		}
		resp.Users = append(resp.Users, TenantUsersResponseUsersInner{
			Username:      user.Username,
			Role:          user.Role,
			Status:        UserStatus(user.Status.String()),
			UserID:        user.ID.String(),
			CognitoID:     aws.ToString(u.Username),
			ProfileFields: convertProfileFields(user),
		})
	}
	return &resp, nil
}

func handleListUsersForTenant(a App, c *gin.Context) error {
	sub, err := perms.SubjectFromContext(c)
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
	ten, err := db.ResolveTenant(c.Request.Context(), a.DBCtx(), th)
	if err != nil {
		return apperr.NewNotFound(err.Error())
	}
	resp, err := getUsersForTenant(a, c, ten, sub)
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, resp)
	return nil
}
