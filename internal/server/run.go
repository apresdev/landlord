package server

import (
	"fmt"
	"net/http"
	"os"

	"github.com/apresdev/landlord/internal/db"

	"github.com/apresdev/landlord/internal/rest"
	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/apresdev/landlord/internal/appenv"

	"github.com/a-h/templ"

	"go.uber.org/zap"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/gin-gonic/gin"
)

func Run() error {
	r, err := engine()
	if err != nil {
		return err
	}

	cs := MustClients()
	a := &app{clients: cs}
	rest.AddRestRoutes(a, r)

	r.GET("/", func(c *gin.Context) {
		if err := handleRoot(c, cs.DDB(), cs.Cfg(), cs.Validator()); err != nil {
			c.String(http.StatusInternalServerError, "Error handling root: %v", err)
		}
	})

	r.GET("/help", func(c *gin.Context) {
		render(c, helpPage(appenv.AppName(), appenv.AppAdminEmail(), mailToAdminLink()))
	})

	r.GET("/login", func(c *gin.Context) {
		username, _ := c.Cookie(LandlordUsernameCookie)
		render(c, loginPage(appenv.AppName(), username))
	})
	r.POST("/login/submit", func(c *gin.Context) {
		if err := handleLoginSubmit(a, c); err != nil {
			c.String(http.StatusInternalServerError, "Error logging in: %v", err)
		}
	})

	r.GET("/new-password", func(c *gin.Context) {
		username, _ := c.Cookie(LandlordUsernameCookie)
		render(c, newPasswordPage(appenv.AppName(), username))
	})
	r.POST("/new-password/submit", func(c *gin.Context) {
		if err := handleNewPasswordSubmit(a, c); err != nil {
			c.String(http.StatusInternalServerError, "Error setting new password: %v", err)
		}
	})

	r.GET("/recovery", func(c *gin.Context) {
		username, _ := c.Cookie(LandlordUsernameCookie)
		render(c, recoveryPage(appenv.AppName(), username))
	})
	r.POST("/recovery/submit", func(c *gin.Context) {
		if err := handleRecoverySubmit(a, c); err != nil {
			c.String(http.StatusInternalServerError, "Error handling recovery: %v", err)
		}
	})

	r.GET("/reset-password", func(c *gin.Context) {
		username, _ := c.Cookie(LandlordUsernameCookie)
		errMsg, _ := c.Cookie(LandlordLastError)
		render(c, resetPasswordPage(appenv.AppName(), username, errMsg))
	})
	r.POST("/reset-password/submit", func(c *gin.Context) {
		if err := handleResetPassword(a, c); err != nil {
			c.String(http.StatusInternalServerError,
				"Error resetting password: %v", err)
		}
	})

	r.GET("/logout", func(c *gin.Context) {
		clearLandlordCookies(c, appenv.AppURL())
		c.Redirect(http.StatusFound, "/")
	})
	listenAddr, err := parseListenAddr(os.Getenv("LANDLORD_LISTEN_ADDR"))
	if err != nil {
		return err
	}
	if listenAddr.UseTLS {
		return runTLS(listenAddr.Addr, r)
	} else {
		return runLocal(listenAddr.Addr, r)
	}
}

func runTLS(addr string, r *gin.Engine) error {
	certPath := os.Getenv("LANDLORD_CERT_PATH")
	if certPath == "" {
		return fmt.Errorf("LANDLORD_CERT_PATH must be set")
	}
	keyPath := os.Getenv("LANDLORD_PRIVATE_KEY_PATH")
	if keyPath == "" {
		return fmt.Errorf("LANDLORD_PRIVATE_KEY_PATH must be set")
	}
	if err := r.RunTLS(addr, certPath, keyPath); err != nil {
		return err
	}
	return nil
}

func runLocal(addr string, r *gin.Engine) error {
	if err := r.Run(addr); err != nil {
		return err
	}
	return nil
}

func render(c *gin.Context, t templ.Component) {
	if err := t.Render(c.Request.Context(), c.Writer); err != nil {
		c.String(http.StatusInternalServerError,
			"Error rendering page: %v", err)
		zap.L().Error("Error rendering page", zap.Error(err))
	}
}

type app struct {
	clients Clients
}

func (a *app) Cfg() aws.Config {
	return a.clients.Cfg()
}

func (a *app) Sts() authx.AWSSTS {
	return a.clients.Sts()
}

func (a *app) CogIdp() authx.AWSCognitoIdentityProvider {
	return a.clients.CogIdp()
}

func (a *app) Validator() authx.TokenValidator {
	return a.clients.Validator()
}

func (a *app) DDB() db.AWSDynamoDB {
	return a.clients.DDB()
}

func clientID(c *gin.Context) authx.OpaqueClientID {
	return authx.OpaqueClientID(c.ClientIP())
}
