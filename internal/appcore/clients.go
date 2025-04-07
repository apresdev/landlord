package appcore

import (
	"context"

	"github.com/google/uuid"

	"github.com/apresdev/landlord/internal/appenv"

	"github.com/apresdev/landlord/internal/jwx"

	"github.com/apresdev/landlord/internal/db"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
)

type Clients interface {
	Settings() appenv.Settings
	Ctx() context.Context
	Cfg() aws.Config
	CogIdp() authx.AWSCognitoIdentityProvider
	Validator() jwx.TokenValidator
	DDB() db.AWSDynamoDB
	DBCtx() *db.Context
	RootTenantID() uuid.UUID
}

type clients struct {
	settings     appenv.Settings
	ctx          context.Context
	cfg          aws.Config
	cogIdp       authx.AWSCognitoIdentityProvider
	validator    jwx.TokenValidator
	dynamo       db.AWSDynamoDB
	dbCtx        *db.Context
	rootTenantID uuid.UUID
}

func (c *clients) Settings() appenv.Settings {
	return c.settings
}

func (c *clients) Ctx() context.Context {
	return c.ctx
}

func (c *clients) Cfg() aws.Config {
	return c.cfg
}

func (c *clients) CogIdp() authx.AWSCognitoIdentityProvider {
	return c.cogIdp
}

func (c *clients) Validator() jwx.TokenValidator {
	return c.validator
}

func (c *clients) DBCtx() *db.Context {
	return c.dbCtx
}

func (c *clients) DDB() db.AWSDynamoDB {
	return c.dynamo
}

func (c *clients) RootTenantID() uuid.UUID {
	return c.rootTenantID
}

func newClients(settings appenv.Settings) (*clients, error) {
	ctx := context.Background()

	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &clients{
		settings:  settings,
		ctx:       ctx,
		cfg:       awsCfg,
		cogIdp:    cognitoidentityprovider.NewFromConfig(awsCfg),
		validator: jwx.NewTokenValidator(),
		dynamo:    dynamodb.NewFromConfig(awsCfg),
	}, nil
}

// Must returns concrete implementations of the core interfaces
// to external services used by landlord. Further, it ensures that
// our Dynamo DB table is created and available for use (or panics).
func Must() Clients {
	c, err := newClients(appenv.MustSettings())
	if err != nil {
		panic(err)
	}
	c.dbCtx = db.MustInit(c.Ctx(), c.Settings(), c.DDB())
	c.rootTenantID = authx.MustRootTenant(
		c.Ctx(), c.DBCtx(), c.CogIdp(), c.Settings().UserPoolName())
	return c
}

// MustWithSettings is like Must, but takes a settings object instead of
// using the global singleton. Useful in testing.
func MustWithSettings(settings appenv.Settings) Clients {
	c, err := newClients(settings)
	if err != nil {
		panic(err)
	}
	c.dbCtx = db.MustInit(c.Ctx(), c.Settings(), c.DDB())
	c.rootTenantID = authx.MustRootTenant(
		c.Ctx(), c.DBCtx(), c.CogIdp(), c.Settings().UserPoolName())
	return c
}
