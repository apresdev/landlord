package rest

import (
	"github.com/apresdev/landlord/internal/authx"

	"github.com/apresdev/landlord/internal/apperr"
	"github.com/apresdev/landlord/internal/db"
	"github.com/apresdev/landlord/internal/multi"
	"github.com/apresdev/landlord/internal/perms"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func handleCancelInvitation(a App, c *gin.Context) error {
	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		return err
	}
	tenantID, err := UUIDParam(c, "tenantID")
	if err != nil {
		return err
	}
	userID, err := UUIDParam(c, "userID")
	if err != nil {
		return apperr.NewBadRequest("no userID provided")
	}
	ctx := c.Request.Context()
	ten, err := db.GetTenantByID(ctx, a.DBCtx(), tenantID)
	if err != nil {
		return err
	}
	user, err := db.GetUserInTenantByID(ctx, a.DBCtx(), tenantID, userID)
	if err != nil {
		return err
	}
	return HandleCancelInvitation(a, c, sub, ten, user)
}

func HandleCancelInvitation(
	a App,
	c *gin.Context,
	sub *perms.Subject,
	targetTenant *multi.Tenant,
	targetUser *multi.User) error {

	ctx := c.Request.Context()
	resource := perms.NewResource(
		perms.ResourceKindInvitation, perms.ResourceNameAny, targetTenant, targetUser)
	effect, err := perms.Eval(ctx, a, sub, perms.ActionDelete, resource)
	if err != nil {
		zap.L().Error("could not evaluate permissions", zap.Error(err))
		return apperr.NewUnauthorized("not authorized")
	}
	if effect.IsDenied() {
		zap.L().Info("denied cancel invitation",
			zap.String("subject", sub.String()),
			zap.String("resource", resource.String()))
		return apperr.NewUnauthorized("not authorized")
	}

	if targetUser.Status != multi.UserStatusInvited {
		zap.L().Info("user is not invited",
			zap.String("clientID", string(sub.Cid)),
			zap.String("username", targetUser.Username),
			zap.String("tenantID", targetTenant.ID.String()),
			zap.String("userID", targetUser.ID.String()),
			zap.String("cognitoID", targetUser.CognitoID.String()))
		return apperr.NewBadRequest("cannot cancel invite, user is not invited")
	}

	authx.DeleteCognitoUser(ctx, a.CogIdp(), targetTenant, targetUser)

	return db.DeleteUser(ctx, a.DBCtx(), targetUser)
}
