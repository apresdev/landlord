package server

import (
	"fmt"

	"github.com/a-h/templ"
	"github.com/apresdev/landlord/internal/appenv"
	"github.com/gin-gonic/gin"
)

func mailToAdminLink() templ.SafeURL {
	return templ.URL(fmt.Sprintf("mailto:%s", appenv.AppAdminEmail()))
}

func renderErrorPage(c *gin.Context, errMsg, navMsg, navURL string) {
	render(c, errorPage(
		appenv.AppName(),
		appenv.AppAdminEmail(),
		mailToAdminLink(),
		errMsg,
		navMsg,
		templ.URL(navURL)))
}

func renderCannotBeRegistered(c *gin.Context, email string) {
	renderErrorPage(c,
		fmt.Sprintf("Username address: \"%s\" cannot be registered at this time.", email),
		"Help",
		"/help")
}
