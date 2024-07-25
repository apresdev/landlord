package appenv

import (
	"net/url"
	"os"
	"strings"

	"go.uber.org/zap"

	"github.com/joho/godotenv"
)

var (
	Debug      bool
	LocalStack bool
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
	s := os.Getenv("AWS_PROFILE")
	if s != "" {
		LocalStack = strings.ToLower(s) == "localstack"
	}
	s = os.Getenv("LANDLORD_DEBUG")
	if s != "" {
		Debug = strings.ToLower(s) == "true"
	}
}

var (
	appURL        *url.URL
	appName       string
	appAdminEmail string
)

func AppURL() *url.URL {
	return appURL
}

func AppName() string {
	return appName
}

func AppAdminEmail() string {
	return appAdminEmail
}

func init() {
	u := os.Getenv("LANDLORD_APP_URL")
	if u == "" {
		panicDuringInit("LANDLORD_APP_URL not set")
	}
	var err error
	appURL, err = url.Parse(u)
	if err != nil {
		panicDuringInit("LANDLORD_APP_URL is not a valid URL", zap.Error(err))
	}
	appName = os.Getenv("LANDLORD_APP_NAME")
	if appName == "" {
		panicDuringInit("LANDLORD_APP_NAME not set")
	}
	appAdminEmail = os.Getenv("LANDLORD_APP_ADMIN_EMAIL")
	if appAdminEmail == "" {
		zap.L().Fatal("LANDLORD_APP_ADMIN_EMAIL not set")
	}
}
