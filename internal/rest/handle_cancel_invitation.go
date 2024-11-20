package rest

import (
	"fmt"

	"github.com/apresdev/landlord/internal/authx"

	"github.com/apresdev/landlord/internal/apperr"
	"github.com/apresdev/landlord/internal/db"
	"github.com/apresdev/landlord/internal/multi"
	"github.com/apresdev/landlord/internal/perms"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func handleCancelInvitation(app App, c *gin.Context) error {
	tenantID, err := UUIDParam(c, "tenantID")
	if err != nil {
		return err
	}
	userID, err := UUIDParam(c, "userID")
	if err != nil {
		return apperr.NewBadRequest("no userID provided")
	}
	return HandleCancelInvitation(app, c, tenantID, userID)
}

func HandleCancelInvitation(app App, c *gin.Context, tenantID, userID uuid.UUID) error {
	ctx := c.Request.Context()
	p, err := perms.PrincipalFromContext(ctx)
	if err != nil {
		return err
	}
	ten, err := db.GetTenantByID(ctx, app.DDB(), tenantID)
	if err != nil {
		return err
	}
	user, err := db.GetUserInTenantByID(ctx, app.DDB(), tenantID, userID)
	if err != nil {
		return err
	}
	resource := perms.NewResource(perms.ResourceKindInvitation, perms.ResourceNameAny, ten, user)
	effect, err := perms.Eval(ctx, app, p, perms.OperationDelete, resource)
	if err != nil {
		zap.L().Error("could not evaluate permissions", zap.Error(err))
		return apperr.NewUnauthorized("not authorized")
	}
	if effect.IsDenied() {
		zap.L().Info("denied cancel invitation",
			zap.String("principal", p.String()),
			zap.String("resource", resource.String()))
		return apperr.NewUnauthorized("not authorized")
	}

	if user.Status != multi.UserStatusInvited {
		return apperr.NewBadRequest(
			fmt.Sprintf("cannot cancel invite, user '%s' is not invited", userID.String()))
	}

	authx.DeleteCognitoUser(ctx, app.CogIdp(), ten, user)

	return db.DeleteUser(ctx, app.DDB(), user)
}
