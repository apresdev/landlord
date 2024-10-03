package rest

import (
	"github.com/apresdev/landlord/internal/apperr"
	"github.com/apresdev/landlord/internal/db"
	"github.com/apresdev/landlord/internal/multi"
	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
)

func handleUpdateUserMFA(app App, c *gin.Context) error {
	var req UpdateUserMfaRequest
	if err := c.BindJSON(&req); err != nil {
		return apperr.NewBadRequest(err.Error())
	}
	if req.MfaPolicy == "" {
		return apperr.NewBadRequest("MFA policy is required")
	}
	tenant := c.Param("tenant")
	if tenant == "" {
		return apperr.NewBadRequest("tenant is required")
	}
	username := c.Param("username")
	if username == "" {
		return apperr.NewBadRequest("username is required")
	}
	return HandleUpdateUserMFA(app, c, tenant, username, req)
}

func HandleUpdateUserMFA(
	app App, c *gin.Context, tenantID, username string, req UpdateUserMfaRequest) error {

	rc, err := requesterContext(c)
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
		// TODO: unhandled case (for now) -- turn off MFA for a user
		return apperr.NewBadRequest("cannot remove MFA policy")
	}
	return nil
}
