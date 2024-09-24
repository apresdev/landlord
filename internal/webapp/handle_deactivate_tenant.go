package webapp

import (
	"fmt"

	"github.com/apresdev/landlord/internal/multi"

	"github.com/apresdev/landlord/internal/rest"

	"github.com/gin-gonic/gin"
)

func handleDeactivateTenant(a App, c *gin.Context) error {
	i := c.Param("id")
	if i == "" {
		return fmt.Errorf("no id provided")
	}
	ten, err := rest.HandleSetTenantState(a, c, i, string(multi.TenantStateInactive))
	if err != nil {
		return err
	}
	render(c, tenantFragment(ten))
	return nil
}
