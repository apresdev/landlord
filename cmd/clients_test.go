package cmd_test

import (
	"context"
	"testing"

	"github.com/apresdev/landlord/cmd"
	"github.com/apresdev/landlord/internal/authx"
	"github.com/apresdev/landlord/internal/mocks"
	"github.com/aws/aws-sdk-go-v2/aws"
)

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
	cmd.OverrideClient = cs
	return cs, func() {
		cmd.OverrideClient = nil
	}
}
