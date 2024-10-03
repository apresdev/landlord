package webapp

import (
	"encoding/base64"
	"fmt"
	"io/fs"
	"net/http"
	"strings"

	"github.com/google/uuid"

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

type App interface {
	Cfg() aws.Config
	CogIdp() authx.AWSCognitoIdentityProvider
	Validator() jwx.TokenValidator
	DDB() db.AWSDynamoDB
	Settings() appenv.Settings
	RootTenantID() uuid.UUID
}

func AddRoutes(a App, baseRoutes *gin.Engine) {
	authorizedRoutes := baseRoutes.Group(appPath(""))
	authorizedRoutes.Use(setupContext(a))
	unauthorizedRoutes := baseRoutes.Group(appPath(""))

	// The routes associated with login, logout, and account recovery are
	// unauthorized because they are the routes that result in the user obtaining
	// a token for the authorized routes.

	baseRoutes.GET("/", func(c *gin.Context) {
		redirectToAppRoot(c)
	})

	unauthorizedRoutes.GET("/", func(c *gin.Context) {
		if err := handleRoot(a, c); err != nil {
			c.String(http.StatusInternalServerError, "Error handling root: %v", err)
		}
	})

	unauthorizedRoutes.GET("/help", func(c *gin.Context) {
		render(c, helpPage(
			a.Settings().AppName(),
			a.Settings().AdminEmail(),
			mailToAdminLink(a.Settings().AdminEmail())))
	})

	unauthorizedRoutes.GET("/login", func(c *gin.Context) {
		username, _ := c.Cookie(LandlordUsernameCookie)
		render(c, loginPage(a.Settings().AppName(), username))
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
		otpImageURI := appPath(fmt.Sprintf(
			"/qrcode?uri=%s", base64.URLEncoding.EncodeToString([]byte(otpURI))))
		username, _ := c.Cookie(LandlordUsernameCookie)
		render(c, setupOTPPage(a.Settings().AppName(), otpImageURI, username))
	})
	unauthorizedRoutes.POST("/setup-otp/submit", func(c *gin.Context) {
		if err := handleSetupOTPSubmit(a, c); err != nil {
			c.String(http.StatusInternalServerError, "Error setting up OTP: %v", err)
		}
	})

	unauthorizedRoutes.GET("/otp", func(c *gin.Context) {
		username, _ := c.Cookie(LandlordUsernameCookie)
		render(c, otpCodePage(a.Settings().AppName(), username))
	})
	unauthorizedRoutes.POST("/otp/submit", func(c *gin.Context) {
		if err := handleOTPSubmit(a, c); err != nil {
			c.String(http.StatusInternalServerError, "Error handling OTP: %v", err)
		}
	})

	unauthorizedRoutes.GET("/new-password", func(c *gin.Context) {
		username, _ := c.Cookie(LandlordUsernameCookie)
		render(c, newPasswordPage(a.Settings().AppName(), username))
	})
	unauthorizedRoutes.POST("/new-password/submit", func(c *gin.Context) {
		if err := handleNewPasswordSubmit(a, c); err != nil {
			c.String(http.StatusInternalServerError, "Error setting new password: %v", err)
		}
	})

	unauthorizedRoutes.GET("/recovery", func(c *gin.Context) {
		username, _ := c.Cookie(LandlordUsernameCookie)
		render(c, recoveryPage(a.Settings().AppName(), username))
	})
	unauthorizedRoutes.POST("/recovery/submit", func(c *gin.Context) {
		if err := handleRecoverySubmit(a, c); err != nil {
			c.String(http.StatusInternalServerError, "Error handling recovery: %v", err)
		}
	})

	unauthorizedRoutes.GET("/reset-password", func(c *gin.Context) {
		username, _ := c.Cookie(LandlordUsernameCookie)
		errMsg, _ := c.Cookie(LandlordLastError)
		render(c, resetPasswordPage(a.Settings().AppName(), username, errMsg))
	})
	unauthorizedRoutes.POST("/reset-password/submit", func(c *gin.Context) {
		if err := handleResetPassword(a, c); err != nil {
			c.String(http.StatusInternalServerError,
				"Error resetting password: %v", err)
		}
	})

	unauthorizedRoutes.GET("/logout", func(c *gin.Context) {
		clearLandlordCookies(c, a.Settings().URL())
		redirectToAppRoot(c)
	})

	authorizedRoutes.GET("/admin/dashboard", func(c *gin.Context) {
		if err := handleAdminDashboard(a, c); err != nil {
			c.String(http.StatusInternalServerError,
				"Error handling admin dashboard: %v", err)
		}
	})

	authorizedRoutes.GET("/admin/tenants", func(c *gin.Context) {
		if err := handleAdminTenants(a, c); err != nil {
			c.String(http.StatusInternalServerError,
				"Error handling admin tenants: %v", err)
		}
	})
	authorizedRoutes.GET("/admin/tenants/new", func(c *gin.Context) {
		render(c, newTenantPage(a.Settings().AppName()))
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

	authorizedRoutes.GET("/admin/tenants/:tenantID",
		func(c *gin.Context) {
			if err := handleAdminTenant(a, c); err != nil {
				c.String(http.StatusInternalServerError,
					"Error deactivating tenant: %v", err)
			}
		})
	authorizedRoutes.GET("/admin/tenants/:tenantID/invite",
		func(c *gin.Context) {
			if err := handleAdminInvite(a, c); err != nil {
				c.String(http.StatusInternalServerError,
					"Error sending invite: %v", err)
			}
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
	authorizedRoutes.POST("/admin/tenants/:tenantID/users/:username/mfa",
		func(c *gin.Context) {
			if err := handleUpdateMFA(a, c); err != nil {
				c.String(http.StatusInternalServerError,
					"Error handling MFA: %v", err)
			}
		})

	authorizedRoutes.GET("/admin/profile", func(c *gin.Context) {
		if err := handleAdminProfile(a, c); err != nil {
			c.String(http.StatusInternalServerError,
				"Error handling admin profile: %v", err)
		}
	})

	if appenv.DevMode() {
		unauthorizedRoutes.StaticFS(
			"/static", http.Dir("internal/webapp/static"))
		baseRoutes.StaticFileFS(
			"/favicon.ico", "favicon.ico", http.Dir("internal/webapp/static"))
	} else {
		sfs, err := fs.Sub(staticFS, "static")
		if err != nil {
			zap.L().Fatal("Coder fail: could not initialize static file system", zap.Error(err))
		}
		unauthorizedRoutes.StaticFS(
			"/static", http.FS(sfs))
		baseRoutes.StaticFileFS(
			"/favicon.ico", "static/favicon.ico", http.FS(sfs))
	}
}

func render(c *gin.Context, t templ.Component) {
	if err := t.Render(c.Request.Context(), c.Writer); err != nil {
		c.String(http.StatusInternalServerError,
			"Error rendering page: %v", err)
		zap.L().Error("Error rendering page", zap.Error(err))
	}
}

func clientID(c *gin.Context) jwx.OpaqueClientID {
	return jwx.OpaqueClientID(c.ClientIP())
}

func unauthorizedResponse(app App, c *gin.Context) {
	// This redirect ensures that the user is sent back to the login page
	// if they are in an HTMX context where responses are being handling
	// inline.
	c.Header("HX-Redirect", "/login")
	c.Status(http.StatusUnauthorized)
	// The message here is purposefully vague to avoid leaking information.
	// See the logs for diagnosing the issue.
	render(c, unauthorizedPage(
		app.Settings().AppName(), "Credentials are invalid or expired"))
	c.Abort()
}

func setupContext(app App) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid := clientID(c)
		rawAccessToken, err := c.Cookie(LandlordAccessTokenCookie)
		if rawAccessToken == "" || err != nil {
			zap.L().Info("unauthorized: no access token cookie",
				zap.String("clientID", string(cid)),
				zap.Error(err))
			unauthorizedResponse(app, c)
			return
		}
		username, err := c.Cookie(LandlordUsernameCookie)
		if username == "" || err != nil {
			zap.L().Info("unauthorized: no username cookie",
				zap.String("clientID", string(cid)),
				zap.Error(err))
			unauthorizedResponse(app, c)
			return
		}
		_, ten, err := db.GetUserAndTenant(c.Request.Context(), app.DDB(), username)
		if err != nil {
			zap.L().Info("unauthorized: could not find user",
				zap.String("clientID", string(cid)),
				zap.String("username", username),
				zap.Error(err))
			unauthorizedResponse(app, c)
			return
		}
		up, err := authx.GetUserPoolByNameOrID(
			c.Request.Context(), app.CogIdp(), ten.UserPoolID)
		if err != nil {
			zap.L().Info("unauthorized: could not resolve user pool",
				zap.String("clientID", string(cid)),
				zap.String("tenantID", ten.ID.String()),
				zap.String("userPoolID", ten.UserPoolID),
				zap.Error(err))
			unauthorizedResponse(app, c)
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
			unauthorizedResponse(app, c)
			return
		}
		c.Set(rest.TokenKey, tok)
		c.Set(rest.TenantIDKey, ten.ID.String())
		c.Next()
	}
}

func appPath(s string) string {
	s = strings.TrimPrefix(s, "/")
	return fmt.Sprintf("/ll-auth/app/%s", s)
}

func redirectTo(c *gin.Context, path string) {
	c.Redirect(http.StatusFound, appPath(path))
}

func redirectToLogin(c *gin.Context) {
	redirectTo(c, "login")
}

func redirectToAppRoot(c *gin.Context) {
	redirectTo(c, "")
}
