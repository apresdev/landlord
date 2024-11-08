package rest

import (
	"github.com/apresdev/landlord/internal/apperr"
	"github.com/google/uuid"

	"github.com/apresdev/landlord/internal/db"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/gin-gonic/gin"
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
	rc, err := authx.RequesterFromContext(c)
	if err != nil {
		return err
	}
	ten, err := db.ResolveTenant(c.Request.Context(), app.DDB(), rc.Th)
	if err != nil {
		zap.L().Error("error resolving tenant",
			zap.Error(err),
			zap.String("cid", string(rc.Cid)),
			zap.String("tenantID", rc.Th.String()))
		return apperr.NewBadRequest("invalid tenant")
	}
	if !authx.IsAuthorizedForTenant(app.RootTenantID(), ten, rc.Roles) {
		return apperr.NewUnauthorized("not authorized")
	}

	// TODO: check if the caller's role is allowed to cancel invitations

	return authx.CancelInvitation(
		c.Request.Context(), app.DDB(), app.CogIdp(), tenantID, userID)
}
