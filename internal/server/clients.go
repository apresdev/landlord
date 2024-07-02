package server

import (
	"context"
	"testing"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/apresdev/landlord/internal/mocks"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentity"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type Clients interface {
	Ctx() context.Context
	Cfg() aws.Config
	Sts() authx.AWSSTS
	CogId() authx.AWSCognitoIdentity
	CogIdp() authx.AWSCognitoIdentityProvider
	IAM() authx.AWSIAM
	SES() authx.AWSSES
	Validator() authx.TokenValidator
}

type clients struct {
	ctx       context.Context
	cfg       aws.Config
	sts       authx.AWSSTS
	cogId     authx.AWSCognitoIdentity
	cogIdp    authx.AWSCognitoIdentityProvider
	iam       authx.AWSIAM
	ses       authx.AWSSES
	validator authx.TokenValidator
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

func (c *clients) CogId() authx.AWSCognitoIdentity {
	return c.cogId
}

func (c *clients) CogIdp() authx.AWSCognitoIdentityProvider {
	return c.cogIdp
}

func (c *clients) IAM() authx.AWSIAM {
	return c.iam
}

func (c *clients) SES() authx.AWSSES {
	return c.ses
}

func (c *clients) Validator() authx.TokenValidator {
	return c.validator
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
		cogId:     cognitoidentity.NewFromConfig(awsCfg),
		cogIdp:    cognitoidentityprovider.NewFromConfig(awsCfg),
		iam:       iam.NewFromConfig(awsCfg),
		ses:       ses.NewFromConfig(awsCfg),
		validator: authx.NewTokenValidator(),
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

type MockClients struct {
	ctx           context.Context
	cfg           aws.Config
	MockSts       *mocks.AWSSTS
	MockCogId     *mocks.AWSCognitoIdentity
	MockCogIdp    *mocks.AWSCognitoIdentityProvider
	MockIAM       *mocks.AWSIAM
	MockSES       *mocks.AWSSES
	MockValidator *mocks.TokenValidator
}

func (c *MockClients) Ctx() context.Context {
	return c.ctx
}

func (c *MockClients) Cfg() aws.Config {
	return c.cfg
}

func (c *MockClients) Sts() authx.AWSSTS {
	return c.MockSts
}

func (c *MockClients) CogId() authx.AWSCognitoIdentity {
	return c.MockCogId
}

func (c *MockClients) CogIdp() authx.AWSCognitoIdentityProvider {
	return c.MockCogIdp
}

func (c *MockClients) IAM() authx.AWSIAM {
	return c.MockIAM
}

func (c *MockClients) SES() authx.AWSSES {
	return c.MockSES
}

func (c *MockClients) Validator() authx.TokenValidator {
	return c.MockValidator
}

func MockClientsOverride(t *testing.T) (*MockClients, func()) {
	cs := &MockClients{
		ctx:           context.Background(),
		cfg:           aws.Config{Region: "mock-region"},
		MockSts:       mocks.NewAWSSTS(t),
		MockCogId:     mocks.NewAWSCognitoIdentity(t),
		MockCogIdp:    mocks.NewAWSCognitoIdentityProvider(t),
		MockIAM:       mocks.NewAWSIAM(t),
		MockSES:       mocks.NewAWSSES(t),
		MockValidator: mocks.NewTokenValidator(t),
	}
	OverrideClient = cs
	return cs, func() {
		OverrideClient = nil
	}
}
