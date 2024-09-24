package server

import (
	"github.com/apresdev/landlord/internal/appenv"
	"github.com/apresdev/landlord/internal/authx"
	"github.com/apresdev/landlord/internal/db"
	"github.com/apresdev/landlord/internal/jwx"
	"github.com/aws/aws-sdk-go-v2/aws"
)

type app struct {
	clients  Clients
	settings appenv.Settings
}

func newApp(cs Clients) *app {
	return &app{clients: cs, settings: appenv.MustSettings()}
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

func (a *app) Settings() appenv.Settings {
	return a.settings
}
