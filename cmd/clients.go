package cmd

import (
	"context"

	"github.com/apresdev/landlord/internal/jwx"

	"github.com/apresdev/landlord/internal/db"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type Clients interface {
	Ctx() context.Context
	Cfg() aws.Config
	Sts() authx.AWSSTS
	CogIdp() authx.AWSCognitoIdentityProvider
	SES() authx.AWSSES
	Validator() jwx.TokenValidator
	DDB() db.AWSDynamoDB
}

type clients struct {
	ctx       context.Context
	cfg       aws.Config
	sts       authx.AWSSTS
	cogIdp    authx.AWSCognitoIdentityProvider
	ses       authx.AWSSES
	validator jwx.TokenValidator
	dynamo    db.AWSDynamoDB
}

func (c *clients) Ctx() context.Context {
	return c.ctx
}

func (c *clients) Cfg() aws.Config {
	return c.cfg
}

func (c *clients) Sts() authx.AWSSTS {
	return c.sts
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

func (c *clients) DDB() db.AWSDynamoDB {
	return c.dynamo
}

func newClients() (*clients, error) {
	ctx := context.Background()
	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &clients{
		ctx:       ctx,
		cfg:       awsCfg,
		sts:       sts.NewFromConfig(awsCfg),
		cogIdp:    cognitoidentityprovider.NewFromConfig(awsCfg),
		ses:       ses.NewFromConfig(awsCfg),
		validator: jwx.NewTokenValidator(),
		dynamo:    dynamodb.NewFromConfig(awsCfg),
	}, nil
}

var OverrideClient Clients

func MustClients() Clients {
	if OverrideClient != nil {
		return OverrideClient
	}
	c, err := newClients()
	if err != nil {
		panic(err)
	}
	return c
}
