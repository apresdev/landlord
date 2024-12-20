package rest

import (
	"net/http"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/apresdev/landlord/internal/multi"

	"go.uber.org/zap"

	"github.com/apresdev/landlord/internal/perms"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/gin-gonic/gin"
)

func handleRetrieveSubject(app App, c *gin.Context) error {
	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		return err
	}
	user, err := sub.ResolveUser(c.Request.Context(), app)
	if err != nil {
		zap.L().Error("error resolving user", zap.Error(err))
		return apperr.NewUnauthorized("not authorized")
	}
	ten, err := sub.ResolveTenant(c.Request.Context(), app)
	if err != nil {
		zap.L().Error("error resolving tenant", zap.Error(err))
		return apperr.NewUnauthorized("not authorized")
	}
	var resp SubjectResponse
	resp.UserID = user.ID.String()
	resp.Username = sub.Username
	resp.Role = sub.Role.Name
	resp.Status = UserStatus(user.Status.String())
	resp.TenantID = ten.ID.String()
	resp.TenantName = ten.Name
	resp.TenantProfileFields = convertProfileFields(ten)
	resp.UserProfileFields = convertProfileFields(user)

	if sub.IsImpersonating() {
		resp.Impersonating.TenantID = sub.ImpersonateTenant.String()
		resp.Impersonating.UserID = sub.ImpersonateSub.String()
		resp.Impersonating.Email = sub.ImpersonateEmail
		role, err := authx.ParseRoleFromGroup(sub.ImpersonateGroup)
		roleName := multi.RoleGuest
		if err != nil {
			zap.L().Error("error parsing role from group", zap.Error(err))
		} else {
			roleName = role.Name
		}
		resp.Impersonating.Role = roleName
	}

	c.JSON(http.StatusOK, resp)
	return nil
}

func convertProfileFields(prov multi.ProfileFieldProvider) []ProfileFormatField {
	var result []ProfileFormatField
	for _, field := range prov.ProfileFields() {
		result = append(result, ProfileFormatField{
			Name:   field.Name,
			Format: field.Format.String(),
			Value:  field.Value,
		})
	}
	return result
}
