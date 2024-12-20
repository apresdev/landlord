package rest

import (
	"fmt"

	"github.com/apresdev/landlord/internal/apperr"
	"github.com/apresdev/landlord/internal/db"
	"github.com/apresdev/landlord/internal/perms"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
)

func handleUpdateTenantProfileFields(app App, c *gin.Context) error {
	var req UpdateProfileFieldsRequest
	if err := c.BindJSON(&req); err != nil {
		return apperr.NewBadRequest(err.Error())
	}
	tenantID, err := UUIDParam(c, "tenantID")
	if err != nil {
		return apperr.NewBadRequest("invalid tenant ID")
	}
	return HandleUpdateTenantProfileFields(app, c, tenantID, req)
}

func HandleUpdateTenantProfileFields(
	app App, c *gin.Context, tenantID uuid.UUID, req UpdateProfileFieldsRequest) error {

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
	profileDef := targetTenant.GetTenantProfileDefinition()
	for _, update := range req.ProfileFields {
		fieldDef := profileDef.GetFieldDefinition(update.Name)
		if fieldDef == nil {
			return apperr.NewBadRequest("invalid field name: " + update.Name)
		}
		resource := perms.NewResource(
			perms.ResourceKindTenantProfile, fieldDef.Name, targetTenant, nil)
		effect, err := perms.Eval(c.Request.Context(), app, sub, perms.ActionWrite, resource)
		if err != nil {
			zap.L().Error("could not evaluate permissions",
				zap.Error(err),
				zap.String("subject", sub.String()),
				zap.String("resource", resource.String()))
			return apperr.NewUnauthorized("not authorized")
		}
		if effect.IsDenied() {
			zap.L().Info("denied update tenant profile field",
				zap.String("subject", sub.String()),
				zap.String("resource", resource.String()))
			return apperr.NewUnauthorized("not authorized")
		}
		err = fieldDef.ValidateValue(update.Value)
		if err != nil {
			return apperr.NewBadRequestE(
				fmt.Errorf("invalid value for field '%s': %w", update.Name, err))
		}
	}
	for _, update := range req.ProfileFields {
		defn := profileDef.GetFieldDefinition(update.Name)
		if defn == nil {
			return apperr.NewBadRequest("invalid field name: " + update.Name)
		}
		targetTenant.UpdateProfileField(update.Name, defn.Format, update.Value)
	}
	err = db.StoreTenant(c.Request.Context(), app.DDB(), targetTenant)
	if err != nil {
		return err
	}
	return nil
}
