package webapp

import (
	"fmt"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
)

func mailToAdminLink(adminEmail string) templ.SafeURL {
	return templ.URL(fmt.Sprintf("mailto:%s", adminEmail))
}

func renderErrorPage(a App, c *gin.Context, errMsg, navMsg, navURL string) {
	render(c, errorPage(
		a.Settings().AppName(),
		a.Settings().AdminEmail(),
		mailToAdminLink(a.Settings().AdminEmail()),
		errMsg,
		navMsg,
		templ.URL(navURL)))
}

func renderCannotBeRegistered(a App, c *gin.Context, email string) {
	renderErrorPage(a, c,
		fmt.Sprintf("Username address: \"%s\" cannot be registered at this time.", email),
		"Help",
		"/help")
}
