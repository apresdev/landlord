package rest

import (
	"fmt"

	"github.com/apresdev/landlord/internal/perms"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"

	"github.com/apresdev/landlord/internal/multi"
	"github.com/google/uuid"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/apresdev/landlord/internal/db"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func handleUpdateUserStatus(a App, c *gin.Context) error {
	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		return err
	}
	tenantID := c.Param("tenantID")
	if tenantID == "" {
		return apperr.NewBadRequest("no tenantID provided")
	}
	targetTenantID, err := uuid.Parse(tenantID)
	if err != nil {
		zap.L().Error("error parsing tenant handle",
			zap.String("tenantID", tenantID),
			zap.Error(err))
		return apperr.NewBadRequest("invalid tenantID")
	}
	uID := c.Param("userID")
	if uID == "" {
		return apperr.NewBadRequest("no userID provided")
	}
	userID, err := uuid.Parse(uID)
	if err != nil {
		zap.L().Error("error parsing user ID",
			zap.String("userID", uID),
			zap.Error(err))
		return apperr.NewBadRequest("invalid userID")
	}
	targetTenant, err := db.GetTenantByID(c.Request.Context(), a.DBCtx(), targetTenantID)
	if err != nil {
		zap.L().Error("error resolving tenant",
			zap.String("tenantID", tenantID),
			zap.Error(err))
		return apperr.NewBadRequest("invalid tenant")
	}
	targetUser, err := db.GetUserInTenantByID(c.Request.Context(), a.DBCtx(), targetTenant.ID, userID)
	if err != nil {
		zap.L().Info("user not found when setting user status",
			zap.String("tenantID", targetTenant.ID.String()),
			zap.String("userID", userID.String()),
			zap.Error(err))
		return apperr.NewNotFound(fmt.Sprintf("user not found: %s", userID.String()))
	}
	s := c.Param("userStatus")
	if s == "" {
		return apperr.NewBadRequest("no userStatus provided")
	}
	_, err = HandleUpdateUserStatus(a, c, sub, targetTenant, targetUser, s)
	return err
}

func HandleUpdateUserStatus(
	a App,
	c *gin.Context,
	sub *perms.Subject,
	targetTenant *multi.Tenant,
	targetUser *multi.User,
	rawTargetStatus string) (*multi.User, error) {

	targetStatus, err := multi.ParseUserStatus(rawTargetStatus)
	if err != nil {
		return nil, apperr.NewBadRequestE(err)
	}

	resource := perms.NewResource(
		perms.ResourceKindUserStatus, targetUser.Username, targetTenant, targetUser)
	effect, err := perms.Eval(c, a, sub, perms.ActionWrite, resource)
	if err != nil {
		zap.L().Error("could not evaluate permissions", zap.Error(err))
		return nil, apperr.NewUnauthorized("not authorized")
	}
	if effect.IsDenied() {
		zap.L().Info("denied setting user status",
			zap.String("subject", sub.String()),
			zap.String("resource", resource.String()))
		return nil, apperr.NewUnauthorized("not authorized")
	}

	if targetUser.Status == targetStatus {
		zap.L().Info("user status already set",
			zap.String("tenantID", targetTenant.ID.String()),
			zap.String("username", targetUser.Username),
			zap.String("userID", targetUser.ID.String()),
			zap.String("cognitoID", targetUser.CognitoID.String()),
			zap.String("status", targetStatus.String()))
		return targetUser, nil
	}
	if targetStatus == multi.UserStatusInactive {
		_, err := a.CogIdp().AdminDisableUser(c.Request.Context(),
			&cognitoidentityprovider.AdminDisableUserInput{
				UserPoolId: aws.String(targetTenant.UserPoolID),
				Username:   aws.String(targetUser.ID.String()),
			})
		if err != nil {
			zap.L().Error("error disabling user",
				zap.String("tenantID", targetTenant.ID.String()),
				zap.String("username", targetUser.Username),
				zap.String("userID", targetUser.ID.String()),
				zap.String("cognitoID", targetUser.CognitoID.String()),
				zap.Error(err))
			return nil, apperr.NewInternalE(err)
		}
	} else if targetStatus == multi.UserStatusActive {
		_, err := a.CogIdp().AdminEnableUser(c.Request.Context(),
			&cognitoidentityprovider.AdminEnableUserInput{
				UserPoolId: aws.String(targetTenant.UserPoolID),
				Username:   aws.String(targetUser.ID.String()),
			})
		if err != nil {
			zap.L().Error("error enabling user",
				zap.String("tenantID", targetTenant.ID.String()),
				zap.String("userID", targetUser.ID.String()),
				zap.String("username", targetUser.Username),
				zap.String("cognitoID", targetUser.CognitoID.String()),
				zap.Error(err))
			return nil, apperr.NewInternalE(err)
		}
	}

	u, err := db.SetUserStatus(c.Request.Context(), a.DBCtx(), targetUser, targetStatus)
	if err != nil {
		return nil, err
	}

	return u, nil
}
