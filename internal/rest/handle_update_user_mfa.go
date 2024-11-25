package rest

import (
	"github.com/apresdev/landlord/internal/apperr"
	"github.com/apresdev/landlord/internal/db"
	"github.com/apresdev/landlord/internal/multi"
	"github.com/apresdev/landlord/internal/perms"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
)

func handleUpdateUserMFA(app App, c *gin.Context) error {
	var req UpdateUserMfaRequest
	if err := c.BindJSON(&req); err != nil {
		return apperr.NewBadRequest(err.Error())
	}
	tenantID, err := UUIDParam(c, "tenantID")
	if err != nil {
		return apperr.NewBadRequest("invalid tenant ID")
	}
	userID, err := UUIDParam(c, "userID")
	if err != nil {
		return apperr.NewBadRequest("invalid user ID")
	}
	return HandleUpdateUserMFA(app, c, tenantID, userID, req)
}

func HandleUpdateUserMFA(
	app App, c *gin.Context, tenantID, userID uuid.UUID, req UpdateUserMfaRequest) error {

	if req.MfaPolicy == "" {
		return apperr.NewBadRequest("MFA policy is empty")
	}

	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		return err
	}
	targetTenant, err := db.GetTenantByID(c.Request.Context(), app.DDB(), tenantID)
	if err != nil {
		zap.L().Error("error fetching tenant",
			zap.Error(err),
			zap.String("tenantID", tenantID.String()))
		return apperr.NewUnauthorized("not authorized")
	}
	targetUser, err := db.GetUserInTenantByID(c.Request.Context(), app.DDB(), tenantID, userID)
	if err != nil {
		zap.L().Error("error fetching user",
			zap.Error(err),
			zap.String("tenantID", tenantID.String()),
			zap.String("userID", userID.String()))
		return apperr.NewUnauthorized("not authorized")
	}

	resource := perms.NewResource(
		perms.ResourceKindUserMFAPolicy, perms.ResourceNameAny, targetTenant, targetUser)
	effect, err := perms.Eval(c.Request.Context(), app, sub, perms.ActionWrite, resource)
	if err != nil {
		zap.L().Error("could not evaluate permissions",
			zap.Error(err),
			zap.String("subject", sub.String()),
			zap.String("resource", resource.String()))
		return apperr.NewUnauthorized("not authorized")
	}
	if effect.IsDenied() {
		zap.L().Info("denied update user MFA",
			zap.String("subject", sub.String()),
			zap.String("resource", resource.String()))
		return apperr.NewUnauthorized("not authorized")
	}

	if targetUser.Status == multi.UserStatusInactive {
		zap.L().Info("user is inactive cannot change MFA",
			zap.String("tenantID", tenantID.String()),
			zap.String("userID", userID.String()))
		return apperr.NewBadRequest("user is inactive, cannot modify MFA")
	}
	// Transition from MFA policy None to Required initiates the flow to
	// add an MFA token to the targetUser
	if targetUser.MFAPolicy.String() == string(MFAPolicyNone) && req.MfaPolicy == MFAPolicyRequired {
		targetUser.MFAStatus = multi.MFAStatusPending
		targetUser.MFAPolicy = multi.MFAPolicyRequired
		err := db.StoreUser(c.Request.Context(), app.DDB(), targetUser)
		if err != nil {
			zap.L().Error("error storing user",
				zap.Error(err),
				zap.String("tenantID", tenantID.String()),
				zap.String("userID", userID.String()))
			return apperr.NewUnauthorized("not authorized")
		}
	} else if req.MfaPolicy == MFAPolicyNone {
		targetUser.MFAStatus = multi.MFAStatusNone
		targetUser.MFAPolicy = multi.MFAPolicyNone
		err := db.StoreUser(c.Request.Context(), app.DDB(), targetUser)
		if err != nil {
			zap.L().Error("error storing user",
				zap.Error(err),
				zap.String("tenantID", tenantID.String()),
				zap.String("userID", userID.String()))
			return apperr.NewUnauthorized("not authorized")
		}
		_, err = app.CogIdp().AdminSetUserMFAPreference(
			c.Request.Context(),
			&cognitoidentityprovider.AdminSetUserMFAPreferenceInput{
				UserPoolId: aws.String(targetTenant.UserPoolID),
				Username:   aws.String(targetUser.Username),
				SMSMfaSettings: &types.SMSMfaSettingsType{
					Enabled: false, PreferredMfa: false},
				SoftwareTokenMfaSettings: &types.SoftwareTokenMfaSettingsType{
					Enabled: false, PreferredMfa: false},
			})
		if err != nil {
			zap.L().Error("error removing MFA preference",
				zap.Error(err),
				zap.String("tenantID", tenantID.String()),
				zap.String("userID", userID.String()))
			return apperr.NewInternal("could not remove MFA preference")
		}
		zap.L().Info("removed MFA preference",
			zap.String("tenantID", tenantID.String()),
			zap.String("userID", userID.String()))
	}
	return nil
}
