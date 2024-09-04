package server

import (
	"encoding/base64"
	"fmt"
	"io/fs"
	"net/http"
	"os"

	"github.com/apresdev/landlord/internal/apperr"
	"github.com/apresdev/landlord/internal/multi"

	"github.com/apresdev/landlord/internal/jwx"

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
	unauthorizedRoutes, err := engine()
	if err != nil {
		return err
	}

	cs := MustClients()
	a := &app{clients: cs}
	rest.AddRestRoutes(a, unauthorizedRoutes)
	authorizedRoutes := unauthorizedRoutes.Group("/")
	authorizedRoutes.Use(setupContext(a))

	// The routes associated with login, logout, and account recovery are
	// unauthorized because they are the routes that resut in the user obtaining
	// a token for the authorized routes.

	unauthorizedRoutes.GET("/", func(c *gin.Context) {
		if err := handleRoot(c, cs.DDB(), cs.Cfg(), cs.Validator()); err != nil {
			c.String(http.StatusInternalServerError, "Error handling root: %v", err)
		}
	})

	unauthorizedRoutes.GET("/help", func(c *gin.Context) {
		render(c, helpPage(appenv.AppName(), appenv.AppAdminEmail(), mailToAdminLink()))
	})

	unauthorizedRoutes.GET("/login", func(c *gin.Context) {
		username, _ := c.Cookie(LandlordUsernameCookie)
		render(c, loginPage(appenv.AppName(), username))
	})
	unauthorizedRoutes.POST("/login/submit", func(c *gin.Context) {
		if err := handleLoginSubmit(a, c); err != nil {
			c.String(http.StatusInternalServerError, "Error logging in: %v", err)
		}
	})

	unauthorizedRoutes.GET("/qrcode", func(c *gin.Context) {
		if err := handleQRCode(a, c); err != nil {
			c.String(http.StatusInternalServerError, "Error handling QR code: %v", err)
		}
	})

	unauthorizedRoutes.GET("/setup-otp", func(c *gin.Context) {
		otpURI, _ := c.Cookie(LandlordOTPURI)
		otpImageURI := fmt.Sprintf(
			"/qrcode?uri=%s", base64.URLEncoding.EncodeToString([]byte(otpURI)))
		username, _ := c.Cookie(LandlordUsernameCookie)
		render(c, setupOTPPage(appenv.AppName(), otpImageURI, username))
	})
	unauthorizedRoutes.POST("/setup-otp/submit", func(c *gin.Context) {
		if err := handleSetupOTPSubmit(a, c); err != nil {
			c.String(http.StatusInternalServerError, "Error setting up OTP: %v", err)
		}
	})

	unauthorizedRoutes.GET("/otp", func(c *gin.Context) {
		username, _ := c.Cookie(LandlordUsernameCookie)
		render(c, otpCodePage(appenv.AppName(), username))
	})
	unauthorizedRoutes.POST("/otp/submit", func(c *gin.Context) {
		if err := handleOTPSubmit(a, c); err != nil {
			c.String(http.StatusInternalServerError, "Error handling OTP: %v", err)
		}
	})

	unauthorizedRoutes.GET("/new-password", func(c *gin.Context) {
		username, _ := c.Cookie(LandlordUsernameCookie)
		render(c, newPasswordPage(appenv.AppName(), username))
	})
	unauthorizedRoutes.POST("/new-password/submit", func(c *gin.Context) {
		if err := handleNewPasswordSubmit(a, c); err != nil {
			c.String(http.StatusInternalServerError, "Error setting new password: %v", err)
		}
	})

	unauthorizedRoutes.GET("/recovery", func(c *gin.Context) {
		username, _ := c.Cookie(LandlordUsernameCookie)
		render(c, recoveryPage(appenv.AppName(), username))
	})
	unauthorizedRoutes.POST("/recovery/submit", func(c *gin.Context) {
		if err := handleRecoverySubmit(a, c); err != nil {
			c.String(http.StatusInternalServerError, "Error handling recovery: %v", err)
		}
	})

	unauthorizedRoutes.GET("/reset-password", func(c *gin.Context) {
		username, _ := c.Cookie(LandlordUsernameCookie)
		errMsg, _ := c.Cookie(LandlordLastError)
		render(c, resetPasswordPage(appenv.AppName(), username, errMsg))
	})
	unauthorizedRoutes.POST("/reset-password/submit", func(c *gin.Context) {
		if err := handleResetPassword(a, c); err != nil {
			c.String(http.StatusInternalServerError,
				"Error resetting password: %v", err)
		}
	})

	unauthorizedRoutes.GET("/logout", func(c *gin.Context) {
		clearLandlordCookies(c, appenv.AppURL())
		c.Redirect(http.StatusFound, "/")
	})

	authorizedRoutes.GET("/admin/home", func(c *gin.Context) {
		if err := handleAdminHome(a, c); err != nil {
			c.String(http.StatusInternalServerError,
				"Error creating a new tenant: %v", err)
		}

	})

	authorizedRoutes.GET("/admin/tenants", func(c *gin.Context) {
		tens, err := db.GetAllTenants(cs.Ctx(), cs.DDB())
		if err != nil {
			c.String(http.StatusInternalServerError, "Error getting tenants: %v", err)
			return
		}
		render(c, tenantsPage(appenv.AppName(), tens))
	})
	authorizedRoutes.GET("/admin/tenants/new", func(c *gin.Context) {
		render(c, newTenantPage(appenv.AppName()))
	})
	authorizedRoutes.POST("/admin/tenants/new/submit", func(c *gin.Context) {
		if err := handleNewTenant(a, c); err != nil {
			c.String(http.StatusInternalServerError,
				"Error creating a new tenant: %v", err)
		}
	})
	authorizedRoutes.POST("/admin/tenants/activate/:id", func(c *gin.Context) {
		if err := handleActivateTenant(a, c); err != nil {
			c.String(http.StatusInternalServerError,
				"Error deactivating tenant: %v", err)
		}
	})
	authorizedRoutes.POST("/admin/tenants/deactivate/:id", func(c *gin.Context) {
		if err := handleDeactivateTenant(a, c); err != nil {
			c.String(http.StatusInternalServerError,
				"Error deactivating tenant: %v", err)
		}
	})

	authorizedRoutes.GET("/admin/tenants/:tenantID", func(c *gin.Context) {
		id := c.Param("tenantID")
		th, err := multi.ParseTenantHandle(id)
		if err != nil {
			c.String(http.StatusBadRequest, "Error parsing tenant ID: %v", err)
			return
		}
		ten, err := db.ResolveTenant(cs.Ctx(), cs.DDB(), th)
		if err != nil && !apperr.IsNotFound(err) {
			c.String(http.StatusNotFound, "Error getting tenant: %v", err)
			return
		}
		if err != nil {
			c.String(http.StatusInternalServerError, "Error getting tenants: %v", err)
			return
		}
		users, err := db.GetUsersForTenant(cs.Ctx(), cs.DDB(), ten.ID)
		if err != nil {
			c.String(http.StatusInternalServerError, "Error getting users: %v", err)
			return
		}
		render(c, tenantPage(appenv.AppName(), ten, users))
	})
	authorizedRoutes.GET("/admin/tenants/:tenantID/invite",
		func(c *gin.Context) {
			id := c.Param("tenantID")
			th, err := multi.ParseTenantHandle(id)
			if err != nil {
				c.String(http.StatusBadRequest, "Error parsing tenant ID: %v", err)
				return
			}
			ten, err := db.ResolveTenant(cs.Ctx(), cs.DDB(), th)
			if err != nil && !apperr.IsNotFound(err) {
				c.String(http.StatusNotFound, "Error getting tenant: %v", err)
				return
			}
			// Okay for the `username` to be empty. Simply provides a default value
			// for the UI in situations where we navigate from a place with a known
			// username.
			username := c.Query("username")
			render(c, inviteUserPage(
				appenv.AppName(), id, username, ten.Roles, multi.MFAPolicyStrings()))
		})
	authorizedRoutes.POST("/admin/tenants/:tenantID/invite/submit",
		func(c *gin.Context) {
			if err := handleInviteUser(a, c); err != nil {
				c.String(http.StatusInternalServerError,
					"Error creating a new tenant: %v", err)
			}
		})
	authorizedRoutes.POST("/admin/tenants/:tenantID/users/:username/activate",
		func(c *gin.Context) {
			if err := handleActivateUser(a, c); err != nil {
				c.String(http.StatusInternalServerError,
					"Error deactivating user: %v", err)
			}
		})
	authorizedRoutes.POST("/admin/tenants/:tenantID/users/:username/deactivate",
		func(c *gin.Context) {
			if err := handleDeactivateUser(a, c); err != nil {
				c.String(http.StatusInternalServerError,
					"Error deactivating user: %v", err)
			}
		})
	authorizedRoutes.POST("/admin/tenants/:tenantID/users/:username/cancel-invite",
		func(c *gin.Context) {
			if err := handleCancelInvitation(a, c); err != nil {
				c.String(http.StatusInternalServerError,
					"Error cancelling invite: %v", err)
			}
		})

	if appenv.DevMode() {
		unauthorizedRoutes.StaticFS(
			"/static", http.Dir("internal/server/static"))
		unauthorizedRoutes.StaticFileFS(
			"/favicon.ico", "favicon.ico", http.Dir("internal/server/static"))
	} else {
		sfs, err := fs.Sub(staticFS, "static")
		if err != nil {
			return err
		}
		unauthorizedRoutes.StaticFS(
			"/static", http.FS(sfs))
		unauthorizedRoutes.StaticFileFS(
			"/favicon.ico", "static/favicon.ico", http.FS(sfs))
	}
	listenAddr, err := parseListenAddr(os.Getenv("LANDLORD_LISTEN_ADDR"))
	if err != nil {
		return err
	}
	if listenAddr.UseTLS {
		return runTLS(listenAddr.Addr, unauthorizedRoutes)
	} else {
		return runLocal(listenAddr.Addr, unauthorizedRoutes)
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

func (a *app) Validator() jwx.TokenValidator {
	return a.clients.Validator()
}

func (a *app) DDB() db.AWSDynamoDB {
	return a.clients.DDB()
}

func clientID(c *gin.Context) jwx.OpaqueClientID {
	return jwx.OpaqueClientID(c.ClientIP())
}

func setupContext(app *app) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid := clientID(c)
		rawAccessToken, err := c.Cookie(LandlordAccessTokenCookie)
		if rawAccessToken == "" || err != nil {
			zap.L().Info("no access token found",
				zap.String("clientID", string(cid)),
				zap.Error(err))
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		username, err := c.Cookie(LandlordUsernameCookie)
		if username == "" || err != nil {
			zap.L().Info("no username found",
				zap.String("clientID", string(cid)),
				zap.Error(err))
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		_, ten, err := db.GetUserAndTenant(
			c.Request.Context(), app.DDB(), username)
		if err != nil {
			zap.L().Info("unauthorized: could not find user",
				zap.String("clientID", string(cid)),
				zap.String("username", username),
				zap.Error(err))
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		up, err := authx.GetUserPoolByNameOrID(
			c.Request.Context(), app.CogIdp(), ten.UserPoolID)
		if err != nil {
			zap.L().Info("unauthorized: could not resolve user pool",
				zap.String("clientID", string(cid)),
				zap.String("tenantID", ten.ID.String()),
				zap.Error(err))
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		validationURL := jwx.ValidationURL(app.Cfg().Region, aws.ToString(up.Id))
		tok, err := app.Validator().ValidateToken(
			c.Request.Context(), cid, validationURL, rawAccessToken)
		if err != nil {
			zap.L().Info("unauthorized: could not validate token",
				zap.String("clientID", string(cid)),
				zap.String("tenantID", ten.ID.String()),
				zap.Error(err))
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		c.Set(rest.TokenKey, tok)
		c.Set(rest.TenantIDKey, ten.ID.String())
		c.Next()
	}
}
