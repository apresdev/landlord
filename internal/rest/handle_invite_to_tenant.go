package rest

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/apresdev/landlord/internal/db"
	"github.com/apresdev/landlord/internal/multi"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func handleInviteToTenant(app App, c *gin.Context) error {
	var req InviteRequest
	if err := c.BindJSON(&req); err != nil {
		return apperr.NewBadRequest(err.Error())
	}
	tenantID, err := uuid.Parse(c.Param("tenantID"))
	if err != nil {
		return apperr.NewBadRequest("invalid tenant ID")
	}
	if err := HandleInviteToTenant(app, c, tenantID, req); err != nil {
		return err
	}
	c.JSON(http.StatusNoContent, nil)
	return nil
}

func HandleInviteToTenant(app App, c *gin.Context, tenantID uuid.UUID, req InviteRequest) error {
	ten, err := db.GetTenantByID(c.Request.Context(), app.DDB(), tenantID)
	if err != nil {
		return err
	}
	mfaPol, err := multi.ParseMFAPolicy(string(req.MfaPolicy))
	if err != nil {
		return apperr.NewBadRequest(err.Error())
	}
	err = authx.SendInvite(
		c.Request.Context(), app.DDB(), app.CogIdp(), ten, req.Username, req.Role, mfaPol)
	if err != nil {
		zap.L().Error("could not send invite",
			zap.String("username", req.Username),
			zap.String("role", req.Role),
			zap.Error(err))
		return apperr.NewBadRequest(err.Error())
	}
	return nil
}
