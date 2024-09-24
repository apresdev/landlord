package appenv

import (
	"net/url"
	"os"
	"strings"

	"go.uber.org/zap"

	"github.com/joho/godotenv"
)

var (
	devMode       bool
	devModeSet    bool
	localStack    bool
	localStackSet bool
)

func loadIfExists(filename string) error {
	if _, err := os.Stat(filename); err == nil {
		return godotenv.Load(filename)
	}
	return nil
}

func init() {
	// See
	// https://github.com/joho/godotenv?tab=readme-ov-file#precedence--conventions
	// for a discussion of .env precedence.
	env := os.Getenv("LANDLORD_ENV")
	if "" == env {
		env = "development"
	}
	err := loadIfExists(".env." + env + ".local")
	if err != nil {
		panicDuringInit("error loading .env."+env+".local", zap.Error(err))
	}
	if "test" != env {
		err = loadIfExists(".env.local")
		if err != nil {
			panicDuringInit("error loading .env.local", zap.Error(err))
		}
	}
	err = loadIfExists(".env." + env)
	if err != nil {
		panicDuringInit("error loading .env."+env, zap.Error(err))
	}
	err = loadIfExists(".env")
	if err != nil {
		panicDuringInit("error loading .env", zap.Error(err))
	}
	if DevMode() {
		MustConfigureDevelopmentLogging()
	}
}

// DevMode returns true if the LANDLORD_DEV_MODE environment variable is set to
// "true". This is used to determine whether to enable development mode features.
// See README.md for more information on development mode features.
func DevMode() bool {
	if !devModeSet {
		s := os.Getenv("LANDLORD_DEV_MODE")
		devMode = strings.ToLower(s) == "true"
		devModeSet = true
	}
	return devMode
}

// LocalStack returns true if the AWS_PROFILE environment variable is set to
// "localstack". This is used to determine whether to use the local stack for
// testing.
func LocalStack() bool {
	if !localStackSet {
		s := os.Getenv("AWS_PROFILE")
		localStack = strings.ToLower(s) == "localstack"
		localStackSet = true
	}
	return localStack
}

// OverrideLocalStack sets the local stack mode to v and returns a function that
// resets the local stack mode to its previous value. Used in unit tests
// that poke around with the development environment.
func OverrideLocalStack(v bool) func() {
	current := LocalStack()
	if current == v {
		return func() {}
	}
	localStack = v
	return func() {
		localStack = current
	}
}

type RouteSetting int

const (
	RouteSettingBoth RouteSetting = iota
	RouteSettingRest
	RouteSettingWebApp
)

type Settings struct {
	url        *url.URL
	appName    string
	adminEmail string
	routes     RouteSetting
}

func (s Settings) URL() *url.URL {
	return s.url
}

func (s Settings) AppName() string {
	return s.appName
}

func (s Settings) AdminEmail() string {
	return s.adminEmail
}

func (s Settings) Routes() RouteSetting {
	return s.routes
}

func MustSettings() Settings {
	u := os.Getenv("LANDLORD_APP_URL")
	if u == "" {
		zap.L().Fatal("LANDLORD_APP_URL not set")
	}
	up, err := url.Parse(u)
	if err != nil {
		zap.L().Fatal("LANDLORD_APP_URL is not a valid URL", zap.Error(err))
	}
	n := os.Getenv("LANDLORD_APP_NAME")
	if n == "" {
		zap.L().Fatal("LANDLORD_APP_NAME not set")
	}
	e := os.Getenv("LANDLORD_APP_ADMIN_EMAIL")
	if e == "" {
		zap.L().Fatal("LANDLORD_APP_ADMIN_EMAIL not set")
	}
	r := os.Getenv("LANDLORD_APP_ROUTES")
	rs := RouteSettingBoth
	if r != "" {
		switch r {
		case "both":
			rs = RouteSettingBoth
		case "rest":
			rs = RouteSettingRest
		case "webapp":
			rs = RouteSettingWebApp
		default:
			zap.L().Fatal("LANDLORD_APP_ROUTES must be one of 'both', 'rest', or 'webapp'")
		}
	}
	return Settings{url: up, appName: n, adminEmail: e, routes: rs}
}
