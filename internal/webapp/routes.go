package webapp

import (
	"embed"
	"encoding/base64"
	"fmt"
	"io/fs"
	"net/http"
	"strings"

	"github.com/apresdev/landlord/internal/perms"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/google/uuid"

	"github.com/apresdev/landlord/internal/jwx"

	"github.com/apresdev/landlord/internal/db"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/apresdev/landlord/internal/appenv"

	"github.com/a-h/templ"

	"go.uber.org/zap"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/gin-gonic/gin"
)

//go:embed static/*
var staticFS embed.FS

type App interface {
	Cfg() aws.Config
	CogIdp() authx.AWSCognitoIdentityProvider
	Validator() jwx.TokenValidator
	DDB() db.AWSDynamoDB
	Settings() appenv.Settings
	RootTenantID() uuid.UUID
}

func mailToAdminLink(adminEmail string) templ.SafeURL {
	return templ.URL(fmt.Sprintf("mailto:%s", adminEmail))
}

type handler func(App, *gin.Context) error

func addRoute(group *gin.RouterGroup, app App, method, path string, h handler) {
	group.Handle(method, path, func(c *gin.Context) {
		if err := h(app, c); err != nil {
			base64ErrMsg := base64.URLEncoding.EncodeToString([]byte(err.Error()))
			c.Header("HX-Redirect", appPath("/admin/dashboard-error?error="+base64ErrMsg))
			c.Status(apperr.ToHTTPStatus(err))
			err2 := render(c, errorPage(
				app.Settings().AppName(),
				app.Settings().AdminEmail(),
				mailToAdminLink(app.Settings().AdminEmail()),
				err.Error(),
				"Return to dashboard",
				templ.URL(appPath("/admin/dashboard"))))
			if err2 != nil {
				// We are already in an error context and have hit another error.
				zap.L().Error("Error rendering error page", zap.Error(err2))
				c.Redirect(http.StatusFound, appPath("/admin/dashboard"))
			}
		}
		c.Abort()
	})
}

type routeAdder struct {
	app   App
	group *gin.RouterGroup
}

func (ra routeAdder) GET(path string, h handler) {
	addRoute(ra.group, ra.app, http.MethodGet, path, h)
}

func (ra routeAdder) POST(path string, h handler) {
	addRoute(ra.group, ra.app, http.MethodPost, path, h)
}

func (ra routeAdder) DELETE(path string, h handler) {
	addRoute(ra.group, ra.app, http.MethodDelete, path, h)
}

func AddRoutes(a App, baseRoutes *gin.Engine) {
	authGroup := baseRoutes.Group(appPath(""))
	authGroup.Use(setupAuthContext(a))
	noAuthGroup := baseRoutes.Group(appPath(""))

	noAuth := routeAdder{app: a, group: noAuthGroup}
	auth := routeAdder{app: a, group: authGroup}

	baseRoutes.GET("/", func(c *gin.Context) {
		redirectToAppRoot(c)
	})

	// The routes associated with login, logout, and account recovery are
	// unauthorized because they are the routes that result in the user obtaining
	// a token for the authorized routes.
	noAuth.GET("/", handleRoot)
	noAuth.GET("/help", handleHelp)
	noAuth.GET("/login", handleLogin)
	noAuth.POST("/login/submit", handleLoginSubmit)
	noAuth.GET("/logout", handleLogoutSubmit)
	noAuth.GET("/qrcode", handleQRCode)
	noAuth.GET("/setup-otp", handleSetupOTP)
	noAuth.POST("/setup-otp/submit", handleSetupOTPSubmit)
	noAuth.GET("/otp", handleOTP)
	noAuth.POST("/otp/submit", handleOTPSubmit)
	noAuth.GET("/new-password", handleNewPassword)
	noAuth.POST("/new-password/submit", handleNewPasswordSubmit)
	noAuth.GET("/recovery", handleRecovery)
	noAuth.POST("/recovery/submit", handleRecoverySubmit)
	noAuth.GET("/reset-password", handleResetPassword)
	noAuth.POST("/reset-password/submit", handleResetPasswordSubmit)

	// The authorized routes are the routes that require a valid token to access.
	// They utilize the setupAuthContext middleware to ensure that the token is valid.
	auth.GET("/admin/dashboard", handleAdminDashboard)
	auth.GET("/admin/dashboard-error", handleAdminDashboardError)
	auth.POST("/admin/clear-impersonation", handleAdminClearImpersonation)
	auth.GET("/admin/tenants", handleAdminTenants)
	auth.GET("/admin/tenant-select", handleAdminTenantSelect)
	auth.GET("/admin/tenants/new", handleAdminNewTenant)
	auth.POST("/admin/tenants/new/submit", handleAdminNewTenantSubmit)
	auth.POST("/admin/tenants/activate/:id", handleAdminActivateTenantSubmit)
	auth.POST("/admin/tenants/deactivate/:id", handleAdminDeactivateTenantSubmit)
	auth.GET("/admin/tenants/:tenantID", handleAdminTenant)
	auth.GET("/admin/tenants/:tenantID/:tab", handleAdminTenantTab)
	auth.POST("/admin/tenants/:tenantID/description/edit", handleAdminDescriptionSubmit)
	auth.GET("/admin/tenants/:tenantID/description/edit", handleAdminDescriptionEditForm)
	auth.GET("/admin/tenants/:tenantID/description/form", handleAdminDescriptionForm)
	auth.GET("/admin/tenants/:tenantID/invite-user-form", handleAdminInviteUserForm)
	auth.GET("/admin/tenants/:tenantID/invite-user-button", handleAdminInviteUserButton)
	auth.GET("/admin/tenants/:tenantID/add-role-form", handleAdminAddRoleForm)
	auth.GET("/admin/tenants/:tenantID/add-role-button", handleAdminAddRoleButton)
	auth.POST("/admin/tenants/:tenantID/users", handleAdminInviteUserSubmit)
	auth.POST("/admin/tenants/:tenantID/users/:userID/activate", handleAdminActivateUserSubmit)
	auth.POST("/admin/tenants/:tenantID/users/:userID/deactivate", handleAdminDeactivateUserSubmit)
	auth.POST("/admin/tenants/:tenantID/users/:userID/cancel-invite", handleAdminCancelInviteSubmit)
	auth.POST("/admin/tenants/:tenantID/users/:userID/mfa/enable", handleAdminEnableMFASubmit)
	auth.POST("/admin/tenants/:tenantID/users/:userID/mfa/disable", handleAdminDisableMFASubmit)
	auth.POST("/admin/tenants/:tenantID/users/:userID/reinvite", handleAdminReInviteUserSubmit)
	auth.POST("/admin/tenants/:tenantID/users/:userID/impersonate", handleAdminImpersonateUserSubmit)
	auth.POST("/admin/tenants/:tenantID/roles", handleAdminAddRoleSubmit)
	auth.DELETE("/admin/tenants/:tenantID/roles/:role", handleAdminDeleteRoleSubmit)
	auth.GET("/admin/tenants/:tenantID/users/:userID", handleAdminUser)
	auth.GET("/admin/profile", handleAdminProfile)
	auth.GET("/admin/profile/:tab", handleAdminProfileTab)

	// In dev mode, we serve the static files from the local filesystem to make
	// hot reloading easier during dev. In the production build, we serve via the
	// static filesystem that is bound into the binary.
	if appenv.DevMode() {
		noAuthGroup.StaticFS(
			"/static", http.Dir("internal/webapp/static"))
		baseRoutes.StaticFileFS(
			"/favicon.ico", "favicon.ico", http.Dir("internal/webapp/static"))
	} else {
		sfs, err := fs.Sub(staticFS, "static")
		if err != nil {
			zap.L().Fatal("Coder fail: could not initialize static file system", zap.Error(err))
		}
		noAuthGroup.StaticFS(
			"/static", http.FS(sfs))
		baseRoutes.StaticFileFS(
			"/favicon.ico", "static/favicon.ico", http.FS(sfs))
	}
}

func render(c *gin.Context, t templ.Component) error {
	if err := t.Render(c.Request.Context(), c.Writer); err != nil {
		zap.L().Error("Error rendering page", zap.Error(err))
		return err
	}
	return nil
}

func clientID(c *gin.Context) jwx.OpaqueClientID {
	return jwx.OpaqueClientID(c.ClientIP())
}

func unauthorizedResponse(app App, c *gin.Context) {
	// This redirect ensures that the user is sent back to the login page
	// if they are in an HTMX context where responses are being handling
	// inline.
	c.Header("HX-Redirect", appPath("/login"))
	c.Status(http.StatusUnauthorized)
	// The message here is purposefully vague to avoid leaking information.
	// See the logs for diagnosing the issue.
	err := render(c, unauthorizedPage(
		app.Settings().AppName(), "Credentials are invalid or expired"))
	if err != nil {
		zap.L().Error("Error rendering unauthorized page", zap.Error(err))
	}
	c.Abort()
}

func setupAuthContext(app App) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid := perms.ClientID(c)
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
		validationURL := jwx.ValidationURL(app.Cfg().Region, ten.UserPoolID)
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
		c.Set(perms.TokenKey, tok)
		c.Set(perms.TenantIDKey, ten.ID.String())
		c.Next()
	}
}

func appPath(s string) string {
	s = strings.TrimPrefix(s, "/")
	return fmt.Sprintf("/ll-auth/app/%s", s)
}

func appURL(s string) templ.SafeURL {
	return templ.URL(appPath(s))
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
