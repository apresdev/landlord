package authx

import (
	"net/url"
	"os"

	"go.uber.org/zap"
)

var redirectURI string
var appDomainHost string

func init() {
	redirectURI = os.Getenv("LANDLORD_REDIRECT_URI")
	if redirectURI == "" {
		zap.L().Fatal("LANDLORD_REDIRECT_URI not set")
	}
	appDomainURI := os.Getenv("LANDLORD_APP_DOMAIN")
	if appDomainURI == "" {
		zap.L().Fatal("LANDLORD_APP_DOMAIN not set")
	}
	u, err := url.Parse(appDomainURI)
	if err != nil {
		zap.L().Fatal("LANDLORD_APP_DOMAIN is not a valid URL", zap.Error(err))
	}
	appDomainHost = u.Host
}

func RedirectURI() string {
	return redirectURI
}

func AppDomainHost() string {
	return appDomainHost
}
