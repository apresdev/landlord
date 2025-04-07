package appenv

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/google/uuid"

	"github.com/apresdev/landlord/internal/apperr"

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
	if env == "" {
		env = "development"
	}
	err := loadIfExists(".env." + env + ".local")
	if err != nil {
		panicDuringInit("error loading .env."+env+".local", zap.Error(err))
	}
	if env != "test" {
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
	appURL       *url.URL
	consoleURL   *url.URL
	appName      string
	adminEmail   string
	routes       RouteSetting
	dbTableName  string
	userPoolName string
}

func (s Settings) AppURL() *url.URL {
	return s.appURL
}

func (s Settings) ConsoleURL() *url.URL {
	return s.consoleURL
}

func (s Settings) AppName() string {
	return s.appName
}

func (s Settings) AdminEmail() string {
	return s.adminEmail
}

func (s Settings) AdminEmailLink() string {
	return fmt.Sprintf("mailto:%s", s.AdminEmail())
}

func (s Settings) Routes() RouteSetting {
	return s.routes
}

func (s Settings) DBTableName() string {
	return s.dbTableName
}

func (s Settings) UserPoolName() string {
	return s.userPoolName
}

func (s Settings) DevMode() bool {
	return DevMode()
}

func mustEnvURL(key string) *url.URL {
	u, err := envURL(key)
	if err != nil {
		zap.L().Fatal("could not parse URL", zap.String("key", key), zap.Error(err))
	}
	return u
}

func envURL(key string) (*url.URL, error) {
	u := os.Getenv(key)
	if u == "" {
		return nil, apperr.NewNotFoundE(fmt.Errorf("environment variable %s is not set", key))
	}
	up, err := url.Parse(u)
	if err != nil {
		return nil, apperr.NewBadRequestE(err)
	}
	return up, nil
}

func MustSettings() Settings {
	appURL := mustEnvURL("LANDLORD_APP_URL")
	// In a test setup, the consoleURL and the appURL can be the same
	consoleURL, err := envURL("LANDLORD_CONSOLE_URL")
	if err != nil && !apperr.IsNotFound(err) {
		zap.L().Fatal("could not parse LANDLORD_CONSOLE_URL", zap.Error(err))
	} else if err != nil {
		consoleURL = appURL
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
	t := os.Getenv("LANDLORD_DB_TABLE_NAME")
	if t == "" {
		zap.L().Fatal("LANDLORD_DB_TABLE_NAME not set")
	}
	u := os.Getenv("LANDLORD_DEFAULT_USER_POOL")
	if u == "" {
		zap.L().Fatal("LANDLORD_DEFAULT_USER_POOL not set")
	}
	return Settings{
		appURL:       appURL,
		consoleURL:   consoleURL,
		appName:      n,
		adminEmail:   e,
		routes:       rs,
		dbTableName:  t,
		userPoolName: u,
	}
}

// MustTestSettings returns a Settings struct with a random table name for testing.
func MustTestSettings() Settings {
	s := MustSettings()
	s.dbTableName = "dynamodb-test-" + uuid.New().String()
	s.userPoolName = "cognito-test-" + uuid.New().String()
	return s
}
