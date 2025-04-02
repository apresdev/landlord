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
	"github.com/aws/aws-sdk-go-v2/service/ses"
)

type Clients interface {
	Settings() appenv.Settings
	Ctx() context.Context
	Cfg() aws.Config
	CogIdp() authx.AWSCognitoIdentityProvider
	SES() authx.AWSSES
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
	ses          authx.AWSSES
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

func (c *clients) SES() authx.AWSSES {
	return c.ses
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

func newClients() (*clients, error) {
	ctx := context.Background()

	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &clients{
		settings:  appenv.MustSettings(),
		ctx:       ctx,
		cfg:       awsCfg,
		cogIdp:    cognitoidentityprovider.NewFromConfig(awsCfg),
		ses:       ses.NewFromConfig(awsCfg),
		validator: jwx.NewTokenValidator(),
		dynamo:    dynamodb.NewFromConfig(awsCfg),
	}, nil
}

// Must returns concrete implementations of the core interfaces
// to external services used by landlord. Further, it ensures that
// our Dynamo DB table is created and available for use (or panics).
func Must() Clients {
	c, err := newClients()
	if err != nil {
		panic(err)
	}
	c.dbCtx = db.MustInit(c.Ctx(), c.Settings(), c.DDB())
	c.rootTenantID = authx.MustRootTenant(c.Ctx(), c.DBCtx(), c.CogIdp())
	return c
}
