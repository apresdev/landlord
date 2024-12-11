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

func handleUpdateUserProfileFields(app App, c *gin.Context) error {
	var req UpdateUserProfileFieldsRequest
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
	return HandleUpdateUserProfileFields(app, c, tenantID, userID, req)
}

func HandleUpdateUserProfileFields(
	app App, c *gin.Context, tenantID, userID uuid.UUID, req UpdateUserProfileFieldsRequest) error {

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

	tenantProfileDefinition := targetTenant.GetProfileDefinition()
	for _, update := range req.Fields {
		defn := tenantProfileDefinition.GetFieldDefinition(update.Name)
		if defn == nil {
			return apperr.NewBadRequest("invalid field name: " + update.Name)
		}
		resource := perms.NewResource(
			perms.ResourceKindUserProfile, defn.Name, targetTenant, targetUser)
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
		err = defn.ValidateValue(update.Value)
		if err != nil {
			return apperr.NewBadRequestE(
				fmt.Errorf("invalid value for field '%s': %w", update.Name, err))
		}
	}
	for _, update := range req.Fields {
		defn := tenantProfileDefinition.GetFieldDefinition(update.Name)
		if defn == nil {
			return apperr.NewBadRequest("invalid field name: " + update.Name)
		}
		targetUser.UpdateProfileField(update.Name, defn.Format, update.Value)
	}
	err = db.StoreUser(c.Request.Context(), app.DDB(), targetUser)
	if err != nil {
		return err
	}
	return nil
}
