package cmd_test

import (
	"context"
	"testing"

	"github.com/apresdev/landlord/internal/db"

	"github.com/apresdev/landlord/cmd"
	"github.com/apresdev/landlord/internal/authx"
	"github.com/apresdev/landlord/internal/mocks"
	"github.com/aws/aws-sdk-go-v2/aws"
)

type MockClients struct {
	ctx           context.Context
	cfg           aws.Config
	MockSts       *mocks.AWSSTS
	MockCogIdp    *mocks.AWSCognitoIdentityProvider
	MockSES       *mocks.AWSSES
	MockValidator *mocks.TokenValidator
	MockDynamo    *mocks.AWSDynamoDB
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

func (c *MockClients) CogIdp() authx.AWSCognitoIdentityProvider {
	return c.MockCogIdp
}

func (c *MockClients) SES() authx.AWSSES {
	return c.MockSES
}

func (c *MockClients) Validator() authx.TokenValidator {
	return c.MockValidator
}

func (c *MockClients) DDB() db.AWSDynamoDB {
	return c.MockDynamo
}

func MockClientsOverride(t *testing.T) (*MockClients, func()) {
	cs := &MockClients{
		ctx:           context.Background(),
		cfg:           aws.Config{Region: "mock-region"},
		MockSts:       mocks.NewAWSSTS(t),
		MockCogIdp:    mocks.NewAWSCognitoIdentityProvider(t),
		MockSES:       mocks.NewAWSSES(t),
		MockValidator: mocks.NewTokenValidator(t),
		MockDynamo:    mocks.NewAWSDynamoDB(t),
	}
	cmd.OverrideClient = cs
	return cs, func() {
		cmd.OverrideClient = nil
	}
}
