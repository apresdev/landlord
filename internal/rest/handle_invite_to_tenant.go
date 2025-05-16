package rest

import (
	"net/http"

	"github.com/apresdev/landlord/internal/perms"

	"github.com/google/uuid"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/apresdev/landlord/internal/db"
	"github.com/apresdev/landlord/internal/multi"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func handleInviteToTenant(a App, c *gin.Context) error {
	var req InviteRequest
	if err := c.BindJSON(&req); err != nil {
		return apperr.NewBadRequest(err.Error())
	}
	tenantID, err := uuid.Parse(c.Param("tenantID"))
	if err != nil {
		return apperr.NewBadRequest("invalid tenant ID")
	}
	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		return err
	}
	targetTenant, err := db.GetTenantByID(c.Request.Context(), a.DBCtx(), tenantID)
	if err != nil {
		return err
	}
	user, err := HandleInviteToTenant(a, c, sub, targetTenant, req)
	if err != nil {
		return err
	}
	invitedUser := TenantUser{
		Username:  req.Username,
		Role:      req.Role,
		Status:    UserStatus(user.Status),
		UserID:    user.ID.String(),
		CognitoID: user.CognitoID.String(),
	}
	c.JSON(http.StatusOK, &InviteResponse{InvitedUser: invitedUser})
	return nil
}

func HandleInviteToTenant(
	a App,
	c *gin.Context,
	sub *perms.Subject,
	targetTenant *multi.Tenant,
	req InviteRequest) (*multi.User, error) {

	resource := perms.NewResource(
		perms.ResourceKindInvitation, req.Role, targetTenant, nil)
	effect, err := perms.Eval(c, a, sub, perms.ActionWrite, resource)
	if err != nil {
		zap.L().Error("could not evaluate permissions", zap.Error(err))
		return nil, apperr.NewUnauthorized("not authorized")
	}
	if effect.IsDenied() {
		zap.L().Info("denied sending invitation",
			zap.String("subject", sub.String()),
			zap.String("resource", resource.String()))
		return nil, apperr.NewUnauthorized("not authorized")
	}

	mfaPol, err := multi.ParseMFAPolicy(string(req.MfaPolicy))
	if err != nil {
		return nil, apperr.NewBadRequest(err.Error())
	}
	overrideUUID := uuid.Nil
	if req.OptionalUserID != "" {
		overrideUUID, err = uuid.Parse(req.OptionalUserID)
		if err != nil {
			return nil, apperr.NewBadRequest("invalid user ID")
		}
	}

	user, err := authx.SendInvite(
		c.Request.Context(),
		a.DBCtx(),
		a.CogIdp(),
		targetTenant,
		req.Username,
		overrideUUID,
		req.Role,
		mfaPol)
	if err != nil {
		zap.L().Error("could not send invite",
			zap.String("username", req.Username),
			zap.String("role", req.Role),
			zap.Error(err))
		return nil, apperr.NewBadRequest(err.Error())
	}
	return user, nil
}
