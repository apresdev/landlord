package webapp

import (
	"embed"
	"encoding/base64"
	"fmt"
	"io/fs"
	"net/http"
	"strings"

	"github.com/apresdev/landlord/internal/multi"
	"github.com/apresdev/landlord/internal/rest"

	"github.com/apresdev/landlord/internal/timers"

	"github.com/apresdev/landlord/internal/perms"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/google/uuid"

	"github.com/apresdev/landlord/internal/jwx"

	"github.com/apresdev/landlord/internal/db"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/apresdev/landlord/internal/appenv"

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

type plainHandler func(App, *gin.Context) error

func addPlainRoute(group *gin.RouterGroup, a App, method, path string, h plainHandler) {
	group.Handle(method, path, func(c *gin.Context) {
		if err := h(a, c); err != nil {
			processHandlerError(a, c, err)
		}
		c.Abort()
	})
}

type plainRoutes struct {
	app   App
	group *gin.RouterGroup
}

func (pr plainRoutes) GET(path string, h plainHandler) {
	addPlainRoute(pr.group, pr.app, http.MethodGet, path, h)
}

func (pr plainRoutes) POST(path string, h plainHandler) {
	addPlainRoute(pr.group, pr.app, http.MethodPost, path, h)
}

func (pr plainRoutes) DELETE(path string, h plainHandler) {
	addPlainRoute(pr.group, pr.app, http.MethodDelete, path, h)
}

type subjectHandler func(App, *gin.Context, *perms.Subject) error

func addSubjectRoute(group *gin.RouterGroup, a App, method, path string, h subjectHandler) {
	group.Handle(method, path, func(c *gin.Context) {
		sub, err := perms.SubjectFromContext(c)
		if err != nil {
			clearLandlordCookies(c, a.Settings().ConsoleURL())
			redirectToLogin(c)
			return
		}
		if err := h(a, c, sub); err != nil {
			processHandlerError(a, c, err)
		}
		c.Abort()
	})
}

type subjectRoutes struct {
	app   App
	group *gin.RouterGroup
}

func (sr subjectRoutes) GET(path string, h subjectHandler) {
	addSubjectRoute(sr.group, sr.app, http.MethodGet, path, h)
}

func (sr subjectRoutes) POST(path string, h subjectHandler) {
	addSubjectRoute(sr.group, sr.app, http.MethodPost, path, h)
}

func (sr subjectRoutes) DELETE(path string, h subjectHandler) {
	addSubjectRoute(sr.group, sr.app, http.MethodDelete, path, h)
}

type tenantHandler func(App, *gin.Context, *perms.Subject, *multi.Tenant) error

func addTenantRoute(group *gin.RouterGroup, a App, method, path string, h tenantHandler) {
	group.Handle(method, path, func(c *gin.Context) {
		sub, err := perms.SubjectFromContext(c)
		if err != nil {
			clearLandlordCookies(c, a.Settings().ConsoleURL())
			redirectToLogin(c)
			return
		}
		tenantID := c.Param("tenantID")
		if tenantID == "" {
			processHandlerError(a, c, apperr.NewBadRequest("no tenantID provided"))
			return
		}
		th, err := multi.ParseTenantHandle(tenantID)
		if err != nil {
			processHandlerError(a, c, apperr.NewBadRequest("no tenantID provided"))
			return
		}
		targetTenant, err := db.ResolveTenant(c.Request.Context(), a.DDB(), th)
		if err != nil {
			processHandlerError(a, c, apperr.NewBadRequest("could not resolve tenant"))
			return
		}
		if err := h(a, c, sub, targetTenant); err != nil {
			processHandlerError(a, c, err)
		}
		c.Abort()
	})
}

type tenantRoutes struct {
	app   App
	group *gin.RouterGroup
}

func (tr tenantRoutes) GET(path string, h tenantHandler) {
	addTenantRoute(tr.group, tr.app, http.MethodGet, path, h)
}

func (tr tenantRoutes) POST(path string, h tenantHandler) {
	addTenantRoute(tr.group, tr.app, http.MethodPost, path, h)
}

func (tr tenantRoutes) DELETE(path string, h tenantHandler) {
	addTenantRoute(tr.group, tr.app, http.MethodDelete, path, h)
}

type userHandler func(App, *gin.Context, *perms.Subject, *multi.Tenant, *multi.User) error

func addUserRoute(group *gin.RouterGroup, a App, method, path string, h userHandler) {
	group.Handle(method, path, func(c *gin.Context) {
		sub, err := perms.SubjectFromContext(c)
		if err != nil {
			clearLandlordCookies(c, a.Settings().ConsoleURL())
			redirectToLogin(c)
			return
		}
		tenantID := c.Param("tenantID")
		if tenantID == "" {
			processHandlerError(a, c, apperr.NewBadRequest("no tenantID provided"))
			return
		}
		th, err := multi.ParseTenantHandle(tenantID)
		if err != nil {
			processHandlerError(a, c, apperr.NewBadRequest("no tenantID provided"))
			return
		}
		targetTenant, err := db.ResolveTenant(c.Request.Context(), a.DDB(), th)
		if err != nil {
			processHandlerError(a, c, apperr.NewBadRequest("could not resolve tenant"))
			return
		}
		userID, err := rest.UUIDParam(c, "userID")
		if err != nil {
			processHandlerError(a, c, apperr.NewBadRequest("invalid user ID"))
			return
		}
		targetUser, err := db.GetUserInTenantByID(
			c.Request.Context(), a.DDB(), targetTenant.ID, userID)
		if err != nil {
			processHandlerError(a, c, apperr.NewBadRequest("could not find user"))
			return
		}
		if err := h(a, c, sub, targetTenant, targetUser); err != nil {
			processHandlerError(a, c, err)
		}
		c.Abort()
	})
}

type userRoutes struct {
	app   App
	group *gin.RouterGroup
}

func (tr userRoutes) GET(path string, h userHandler) {
	addUserRoute(tr.group, tr.app, http.MethodGet, path, h)
}

func (tr userRoutes) POST(path string, h userHandler) {
	addUserRoute(tr.group, tr.app, http.MethodPost, path, h)
}

func (tr userRoutes) DELETE(path string, h userHandler) {
	addUserRoute(tr.group, tr.app, http.MethodDelete, path, h)
}

func processHandlerError(a App, c *gin.Context, err error) {
	base64ErrMsg := base64.URLEncoding.EncodeToString([]byte(err.Error()))
	c.Header("HX-Redirect", appPath("/admin/dashboard-error?error="+base64ErrMsg))
	c.Status(apperr.ToHTTPStatus(err))
	ev := NewErrorPageView(a, err.Error())
	err2 := executeErrorTemplate(c, "error", ev)
	if err2 != nil {
		// We are already in an error context and have hit another error.
		zap.L().Error("Error rendering error page", zap.Error(err2))
		c.Redirect(http.StatusFound, appPath("/admin/dashboard"))
	}
}

func AddRoutes(a App, baseRoutes *gin.Engine) {
	authGroup := baseRoutes.Group(appPath(""))
	authGroup.Use(setupAuthContext(a))
	noAuthGroup := baseRoutes.Group(appPath(""))

	noAuthPlain := plainRoutes{app: a, group: noAuthGroup}
	authPlain := plainRoutes{app: a, group: authGroup}
	authSubject := subjectRoutes{app: a, group: authGroup}
	authTenant := tenantRoutes{app: a, group: authGroup}
	authUser := userRoutes{app: a, group: authGroup}

	baseRoutes.GET("/", func(c *gin.Context) {
		redirectToAppRoot(c)
	})

	// The routes associated with login, logout, and account recovery are
	// unauthorized because they are the routes that result in the user obtaining
	// a token for the authorized routes.
	noAuthPlain.GET("/", handleRoot)
	noAuthPlain.GET("/login", handleLogin)
	noAuthPlain.POST("/login/submit", handleLoginSubmit)
	noAuthPlain.GET("/logout", handleLogoutSubmit)
	noAuthPlain.GET("/qrcode", handleQRCode)
	noAuthPlain.GET("/setup-otp", handleSetupOTP)
	noAuthPlain.POST("/setup-otp/submit", handleSetupOTPSubmit)
	noAuthPlain.GET("/otp", handleOTP)
	noAuthPlain.POST("/otp/submit", handleOTPSubmit)
	noAuthPlain.GET("/new-password", handleNewPassword)
	noAuthPlain.POST("/new-password/submit", handleNewPasswordSubmit)
	noAuthPlain.GET("/recovery", handleRecovery)
	noAuthPlain.POST("/recovery/submit", handleRecoverySubmit)
	noAuthPlain.GET("/reset-password", handleResetPassword)
	noAuthPlain.POST("/reset-password/submit", handleResetPasswordSubmit)

	// The authorized routes are the routes that require a valid token to access.
	// They utilize the setupAuthContext middleware to ensure that the token is valid.
	authPlain.GET("/admin/dashboard", handleAdminDashboard)
	authPlain.GET("/admin/dashboard-error", handleAdminDashboardError)
	authSubject.POST("/admin/clear-impersonation", handleAdminClearImpersonation)
	authSubject.GET("/admin/tenants", handleAdminTenants)
	authPlain.GET("/admin/tenant-ops/create-tenant-button", handleAdminCreateTenantButton)
	authPlain.GET("/admin/tenant-ops/create-tenant-form", handleAdminCreateTenantForm)
	authSubject.POST("/admin/tenant-ops/create-tenant-submit", handleAdminCreateTenantSubmit)
	authTenant.POST("/admin/tenants/:tenantID/activate", handleAdminActivateTenantSubmit)
	authTenant.POST("/admin/tenants/:tenantID/deactivate", handleAdminDeactivateTenantSubmit)
	authTenant.GET("/admin/tenants/:tenantID", handleAdminTenant)
	authTenant.GET("/admin/tenants/:tenantID/tabs/:tab", handleAdminTenantTab)
	authTenant.POST("/admin/tenants/:tenantID/description/edit", handleAdminDescriptionSubmit)
	authTenant.GET("/admin/tenants/:tenantID/description/edit", handleAdminDescriptionEditForm)
	authTenant.GET("/admin/tenants/:tenantID/description/form", handleAdminDescriptionForm)

	authTenant.GET("/admin/tenants/:tenantID/invite-user-form", handleAdminInviteUserForm)
	authTenant.POST("/admin/tenants/:tenantID/invite-user-form", handleAdminInviteUserSubmit)
	authTenant.GET("/admin/tenants/:tenantID/bulk-invite-form", handleBulkInviteForm)
	authTenant.POST("/admin/tenants/:tenantID/bulk-invite-form", handleBulkInviteSubmit)
	authPlain.GET("/admin/tenants/:tenantID/invite-buttons", handleInviteButtons)
	authTenant.GET("/admin/tenants/:tenantID/bulk-invite-template", handleBulkInviteTemplate)

	authUser.POST("/admin/tenants/:tenantID/users/:userID/activate", handleAdminActivateUserSubmit)
	authUser.POST("/admin/tenants/:tenantID/users/:userID/deactivate", handleAdminDeactivateUserSubmit)
	authUser.POST("/admin/tenants/:tenantID/users/:userID/cancel-invite", handleAdminCancelInviteSubmit)
	authUser.POST("/admin/tenants/:tenantID/users/:userID/mfa/enable", handleAdminEnableMFASubmit)
	authUser.POST("/admin/tenants/:tenantID/users/:userID/mfa/disable", handleAdminDisableMFASubmit)
	authUser.POST("/admin/tenants/:tenantID/users/:userID/reinvite", handleAdminReInviteUserSubmit)
	authUser.POST("/admin/tenants/:tenantID/users/:userID/impersonate", handleAdminImpersonateUserSubmit)

	authUser.POST("/admin/tenants/:tenantID/users/:userID/field/edit", handleAdminUserProfileFieldSubmit)
	authUser.GET("/admin/tenants/:tenantID/users/:userID/field/edit", handleAdminUserProfileFieldEditForm)
	authUser.GET("/admin/tenants/:tenantID/users/:userID/field/form", handlerAdminUserProfileFieldForm)

	authUser.POST("/admin/tenants/:tenantID/users/:userID/role/edit", handleAdminUserRoleSubmit)
	authUser.GET("/admin/tenants/:tenantID/users/:userID/role/edit", handleAdminUserRoleEditForm)
	authUser.GET("/admin/tenants/:tenantID/users/:userID/role/form", handlerAdminUserRoleForm)

	authUser.GET("/admin/tenants/:tenantID/users/:userID/tabs/:tab", handleAdminUserTab)

	authTenant.POST("/admin/tenants/:tenantID/field/edit", handleAdminTenantProfileFieldSubmit)
	authTenant.GET("/admin/tenants/:tenantID/field/edit", handleAdminTenantProfileFieldEditForm)
	authTenant.GET("/admin/tenants/:tenantID/field/form", handlerAdminTenantProfileFieldForm)

	authPlain.GET("/admin/tenants/:tenantID/add-role-form", handleAdminAddRoleForm)
	authPlain.GET("/admin/tenants/:tenantID/add-role-button", handleAdminAddRoleButton)

	authTenant.POST("/admin/tenants/:tenantID/roles", handleAdminAddRoleSubmit)
	authTenant.POST("/admin/tenants/:tenantID/roles/:role/impersonate", handleAdminImpersonateRoleSubmit)
	authTenant.DELETE("/admin/tenants/:tenantID/roles/:role", handleAdminDeleteRoleSubmit)
	authPlain.GET("/admin/tenants/:tenantID/add-sub-role-form", handleAdminAddSubRoleForm)
	authPlain.GET("/admin/tenants/:tenantID/add-sub-role-button", handleAdminAddSubRoleButton)
	authTenant.POST("/admin/tenants/:tenantID/sub-roles", handleAdminAddSubRoleSubmit)
	authTenant.DELETE("/admin/tenants/:tenantID/sub-roles/:subrole", handleAdminDeleteSubRoleSubmit)
	authUser.GET("/admin/tenants/:tenantID/users/:userID", handleAdminUser)
	authTenant.GET("/admin/tenants/:tenantID/activity-report", handleAdminActivityReport)
	authTenant.GET("/admin/tenants/:tenantID/summary-report", handleAdminSummaryReport)

	authSubject.GET("/admin/profile", handleAdminProfile)
	authSubject.GET("/admin/profile/:tab", handleAdminProfileTab)

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
	uv := NewUnauthenticatedView(app.Settings()).
		WithErrMsg("Credentials are invalid or expired")
	err := executeUnauthenticatedView(c, "unauthorized", uv)
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
		c.Set(timers.ContextKey, timers.New())
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
