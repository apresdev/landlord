package rest

import (
	"github.com/apresdev/landlord/internal/apperr"
	"github.com/apresdev/landlord/internal/authx"
	"github.com/apresdev/landlord/internal/db"
	"github.com/apresdev/landlord/internal/multi"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
)

func handleUpdateUserMFA(app App, c *gin.Context) error {
	var req UpdateUserMfaRequest
	if err := c.BindJSON(&req); err != nil {
		return apperr.NewBadRequest(err.Error())
	}
	tenant := c.Param("tenant")
	username := c.Param("username")
	return HandleUpdateUserMFA(app, c, tenant, username, req)
}

func HandleUpdateUserMFA(
	app App, c *gin.Context, tenantID, username string, req UpdateUserMfaRequest) error {

	if tenantID == "" {
		return apperr.NewBadRequest("tenant is required")
	}
	if username == "" {
		return apperr.NewBadRequest("username is required")
	}
	if req.MfaPolicy == "" {
		return apperr.NewBadRequest("MFA policy is required")
	}

	rc, err := authx.RequesterFromContext(c)
	if err != nil {
		return err
	}
	// TODO: finer grained controls
	if err := isRequestAuthorizedForRoot(app, c, rc); err != nil {
		return apperr.NewUnauthorized("not authorized")
	}

	user, tenant, err := db.GetUserAndTenant(c.Request.Context(), app.DDB(), username)
	if err != nil {
		zap.L().Error("error fetching user and tenant",
			zap.Error(err),
			zap.String("tenantID", tenantID),
			zap.String("username", username))
		return apperr.NewUnauthorized("not authorized")
	}
	if tenant.ID.String() != tenantID {
		zap.L().Error("tenantID in path does not match tenant found in DB",
			zap.String("username", username),
			zap.String("tenantID", tenantID),
			zap.String("tenantID_DB", tenant.ID.String()))
		return apperr.NewUnauthorized("not authorized")
	}
	if user.Status == multi.UserStatusInactive {
		zap.L().Info("user is inactive cannot change MFA",
			zap.String("tenantID", tenantID),
			zap.String("username", username))
		return apperr.NewBadRequest("user is inactive, cannot modify MFA")
	}
	// Transition from MFA policy None to Required initiates the flow to
	// add an MFA token to the user.
	if user.MFAPolicy.String() == string(MFAPolicyNone) && req.MfaPolicy == MFAPolicyRequired {
		user.Status = multi.UserStatusPendingMFA
		user.MFAPolicy = multi.MFAPolicyRequired
		err := db.StoreUser(c.Request.Context(), app.DDB(), user)
		if err != nil {
			zap.L().Error("error storing user",
				zap.Error(err),
				zap.String("tenantID", tenantID),
				zap.String("username", username))
			return apperr.NewUnauthorized("not authorized")
		}
	} else if req.MfaPolicy == MFAPolicyNone {
		user.MFAPolicy = multi.MFAPolicyNone
		err := db.StoreUser(c.Request.Context(), app.DDB(), user)
		if err != nil {
			zap.L().Error("error storing user",
				zap.Error(err),
				zap.String("tenantID", tenantID),
				zap.String("username", username))
			return apperr.NewUnauthorized("not authorized")
		}
		_, err = app.CogIdp().AdminSetUserMFAPreference(
			c.Request.Context(),
			&cognitoidentityprovider.AdminSetUserMFAPreferenceInput{
				UserPoolId: aws.String(tenant.UserPoolID),
				Username:   aws.String(user.Username),
				SMSMfaSettings: &types.SMSMfaSettingsType{
					Enabled: false, PreferredMfa: false},
				SoftwareTokenMfaSettings: &types.SoftwareTokenMfaSettingsType{
					Enabled: false, PreferredMfa: false},
			})
		if err != nil {
			zap.L().Error("error removing MFA preference",
				zap.Error(err),
				zap.String("tenantID", tenantID),
				zap.String("username", username))
			return apperr.NewInternal("could not remove MFA preference")
		}
		zap.L().Info("removed MFA preference",
			zap.String("tenantID", tenantID),
			zap.String("username", username))
	}
	return nil
}
