package rest

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"

	"github.com/apresdev/landlord/internal/multi"
	"github.com/google/uuid"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/apresdev/landlord/internal/db"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func handleSetUserStatus(app App, c *gin.Context) error {
	tenantID := c.Param("tenantID")
	if tenantID == "" {
		return apperr.NewBadRequest("no tenantID provided")
	}
	uID := c.Param("userID")
	if uID == "" {
		return apperr.NewBadRequest("no userID provided")
	}
	userID, err := uuid.Parse(uID)
	if err != nil {
		zap.L().Error("error parsing user ID",
			zap.Error(err),
			zap.String("userID", uID))
		return apperr.NewBadRequest("invalid userID")
	}
	s := c.Param("userStatus")
	if s == "" {
		return apperr.NewBadRequest("no userStatus provided")
	}
	_, err = HandleSetUserStatus(app, c, tenantID, userID, s)
	return err
}

func HandleSetUserStatus(
	app App, c *gin.Context, tenantID string, userID uuid.UUID, rawTargetStatus string) (
	*multi.User, error) {

	targetTenantID, err := uuid.Parse(tenantID)
	if err != nil {
		zap.L().Error("error parsing tenant handle",
			zap.Error(err),
			zap.String("tenantID", tenantID))
		return nil, apperr.NewBadRequest("invalid tenantID")
	}
	targetStatus, err := multi.ParseUserStatus(rawTargetStatus)
	if err != nil {
		return nil, apperr.NewBadRequestE(err)
	}

	rc, err := authx.RequesterFromContext(c)
	if err != nil {
		return nil, err
	}

	targetTenant, err := db.GetTenantByID(c.Request.Context(), app.DDB(), targetTenantID)
	if err != nil {
		zap.L().Error("error resolving tenant",
			zap.Error(err),
			zap.String("tenantID", tenantID))
		return nil, apperr.NewBadRequest("invalid tenant")
	}
	// TODO: we only allow this operation for members of the root tenant, for now
	// TODO: check if role is able to set user status. Currently all roles within
	// the root tenant are permitted.
	if !authx.IsAuthorizedForTenant(app.RootTenantID(), targetTenant, rc.Roles) {
		zap.L().Info("user not authorized for tenant when setting user status",
			zap.String("clientID", string(rc.Cid)),
			zap.String("tenantID", tenantID),
			zap.String("roles", rc.Roles.String()))
		return nil, apperr.NewUnauthorized("not authorized")
	}

	targetUser, err := db.GetUserInTenantByID(c.Request.Context(), app.DDB(), targetTenant.ID, userID)
	if err != nil {
		zap.L().Info("user not found when setting user status",
			zap.String("userID", userID.String()),
			zap.Error(err))
		return nil, apperr.NewNotFound(fmt.Sprintf("user not found: %s", userID.String()))
	}
	if targetUser.Status == targetStatus {
		zap.L().Info("user status already set",
			zap.String("userID", userID.String()),
			zap.String("username", targetUser.Username),
			zap.String("status", targetStatus.String()))
		return targetUser, nil
	}
	if targetUser.Status == multi.UserStatusInactive {
		_, err := app.CogIdp().AdminDisableUser(c.Request.Context(),
			&cognitoidentityprovider.AdminDisableUserInput{
				UserPoolId: aws.String(targetTenant.UserPoolID),
				Username:   aws.String(userID.String()),
			})
		if err != nil {
			zap.L().Error("error disabling user",
				zap.String("userID", userID.String()),
				zap.String("username", targetUser.Username),
				zap.Error(err))
			return nil, apperr.NewInternalE(err)
		}
	} else if targetUser.Status == multi.UserStatusActive {
		_, err := app.CogIdp().AdminEnableUser(c.Request.Context(),
			&cognitoidentityprovider.AdminEnableUserInput{
				UserPoolId: aws.String(targetTenant.UserPoolID),
				Username:   aws.String(userID.String()),
			})
		if err != nil {
			zap.L().Error("error enabling user",
				zap.String("userID", userID.String()),
				zap.String("username", targetUser.Username),
				zap.Error(err))
			return nil, apperr.NewInternalE(err)
		}
	}

	u, err := db.SetUserStatus(c.Request.Context(), app.DDB(), targetUser, targetStatus)
	if err != nil {
		return nil, err
	}

	return u, nil
}
