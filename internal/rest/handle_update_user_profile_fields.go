package rest

import (
	"fmt"

	"github.com/apresdev/landlord/internal/multi"

	"github.com/apresdev/landlord/internal/apperr"
	"github.com/apresdev/landlord/internal/db"
	"github.com/apresdev/landlord/internal/perms"
	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
)

func handleUpdateUserProfileFields(a App, c *gin.Context) error {
	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		return err
	}
	var req UpdateProfileFieldsRequest
	if err := c.BindJSON(&req); err != nil {
		return apperr.NewBadRequest(err.Error())
	}
	tenantID := c.Param("tenantID")
	if tenantID == "" {
		return apperr.NewBadRequest("no tenantID provided")
	}
	th, err := multi.ParseTenantHandle(tenantID)
	if err != nil {
		return apperr.NewBadRequest("invalid tenantID")
	}
	targetTenant, err := db.ResolveTenant(c.Request.Context(), a.DBCtx(), th)
	if err != nil {
		return err
	}
	userID, err := UUIDParam(c, "userID")
	if err != nil {
		return apperr.NewBadRequest("invalid user ID")
	}
	targetUser, err := db.GetUserInTenantByID(
		c.Request.Context(), a.DBCtx(), targetTenant.ID, userID)
	if err != nil {
		zap.L().Error("error fetching user",
			zap.Error(err),
			zap.String("tenantID", targetTenant.ID.String()),
			zap.String("userID", userID.String()))
		return apperr.NewUnauthorized("not authorized")
	}
	return HandleUpdateUserProfileFields(a, c, sub, targetTenant, targetUser, req)
}

func HandleUpdateUserProfileFields(
	a App,
	c *gin.Context,
	sub *perms.Subject,
	targetTenant *multi.Tenant,
	targetUser *multi.User,
	req UpdateProfileFieldsRequest) error {

	profileDef := multi.GetUserProfileDefinition()
	for _, update := range req.ProfileFields {
		fieldDef := profileDef.GetFieldDefinition(update.Name)
		if fieldDef == nil {
			return apperr.NewBadRequest("invalid field name: " + update.Name)
		}
		resource := perms.NewResource(
			perms.ResourceKindUserProfile, fieldDef.Name, targetTenant, targetUser)
		effect, err := perms.Eval(c.Request.Context(), a, sub, perms.ActionWrite, resource)
		if err != nil {
			zap.L().Error("could not evaluate permissions",
				zap.Error(err),
				zap.String("subject", sub.String()),
				zap.String("resource", resource.String()))
			return apperr.NewUnauthorized("not authorized")
		}
		if effect.IsDenied() {
			zap.L().Info("denied update user profile field",
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
		targetUser.UpdateProfileField(update.Name, defn.Format, update.Value)
	}
	if err := db.StoreUser(c.Request.Context(), a.DBCtx(), targetUser); err != nil {
		return err
	}
	return nil
}
