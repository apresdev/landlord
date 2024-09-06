package server

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
	Cfg() aws.Config
	Sts() authx.AWSSTS
	CogIdp() authx.AWSCognitoIdentityProvider
	SES() authx.AWSSES
	Validator() jwx.TokenValidator
	DDB() db.AWSDynamoDB
}

type clients struct {
	cfg       aws.Config
	sts       authx.AWSSTS
	cogIdp    authx.AWSCognitoIdentityProvider
	ses       authx.AWSSES
	validator jwx.TokenValidator
	ddb       db.AWSDynamoDB
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
	return c.ddb
}

func newClients() (*clients, error) {
	ctx := context.Background()
	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &clients{
		cfg:       awsCfg,
		sts:       sts.NewFromConfig(awsCfg),
		cogIdp:    cognitoidentityprovider.NewFromConfig(awsCfg),
		ses:       ses.NewFromConfig(awsCfg),
		validator: jwx.NewTokenValidator(),
		ddb:       dynamodb.NewFromConfig(awsCfg),
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
