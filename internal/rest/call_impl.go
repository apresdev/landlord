package rest

import (
	"net/http"
	"slices"
	"strings"

	"github.com/apresdev/landlord/internal/jwx"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/apresdev/landlord/internal/db"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type App interface {
	Cfg() aws.Config
	Sts() authx.AWSSTS
	CogIdp() authx.AWSCognitoIdentityProvider
	Validator() jwx.TokenValidator
	DDB() db.AWSDynamoDB
}

const (
	AppContextKey    = "app"
	BaseLandlordPath = "/ll-auth"
)

func pathFor(path string) string {
	return BaseLandlordPath + path
}

var unauthorizedPaths = []string{
	pathFor("/api/login"),
	pathFor("/api/logout"),
	pathFor("/api/recover-password"),
	pathFor("/api/reset-password"),
	pathFor("/api/new-password"),
}

func AddRestRoutes(app App, router *gin.Engine) http.Handler {
	router.Use(func(c *gin.Context) {
		c.Set(AppContextKey, app)
	})
	router.Use(cors.Default()) // Permissive CORS
	authorized := router.Group("/")
	authorized.Use(isAuthorized(app))
	for _, subRoute := range getRoutes() {
		if subRoute.Pattern == "/" {
			continue
		}
		if slices.Contains(unauthorizedPaths, subRoute.Pattern) {
			router.Handle(subRoute.Method, subRoute.Pattern, subRoute.HandlerFunc)
			continue
		}
		authorized.Handle(subRoute.Method, subRoute.Pattern, subRoute.HandlerFunc)
	}
	return router
}

const (
	errCouldNotFetchAppFromContext = "could not fetch app from context"
	errCouldNotConvertContextToApp = "could not convert context to app"
)

func newAPIError(msg string) ApiError {
	result := ApiError{}
	result.Status = APIStatusFailure
	result.Data.Errors = append(result.Data.Errors, FieldError{
		Message: msg,
	})
	return result
}

func callImpl(c *gin.Context, impl func(App, *gin.Context) error) {
	a, ok := c.Get(AppContextKey)
	if !ok {
		zap.L().Error(errCouldNotFetchAppFromContext)
		c.JSON(http.StatusInternalServerError, newAPIError(errCouldNotConvertContextToApp))
		return
	}
	var sb strings.Builder
	_ = c.Request.Header.Write(&sb)
	app, ok := a.(App)
	if !ok {
		zap.L().Error(errCouldNotConvertContextToApp)
		c.JSON(http.StatusInternalServerError, newAPIError(errCouldNotConvertContextToApp))
		return
	}
	err := impl(app, c)
	if err != nil {
		zap.L().Error("error handling request", zap.Error(err))
		c.JSON(apperr.ToHTTPStatus(err), newAPIError(err.Error()))
		c.Abort()
	}
}
