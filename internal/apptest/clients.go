package apptest

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/apresdev/landlord/internal/appenv"

	"github.com/apresdev/landlord/internal/appcore"

	"github.com/apresdev/landlord/internal/jwx"

	"github.com/apresdev/landlord/internal/db"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/apresdev/landlord/internal/mocks"
	"github.com/aws/aws-sdk-go-v2/aws"
)

type MockClients struct {
	ctx           context.Context
	cfg           aws.Config
	MockCogIdp    *mocks.AWSCognitoIdentityProvider
	MockSES       *mocks.AWSSES
	MockValidator *mocks.TokenValidator
	MockDynamo    *mocks.AWSDynamoDB
	settings      appenv.Settings
	rootTenantID  uuid.UUID
}

func (c *MockClients) Ctx() context.Context {
	return c.ctx
}

func (c *MockClients) Cfg() aws.Config {
	return c.cfg
}

func (c *MockClients) CogIdp() authx.AWSCognitoIdentityProvider {
	return c.MockCogIdp
}

func (c *MockClients) SES() authx.AWSSES {
	return c.MockSES
}

func (c *MockClients) Validator() jwx.TokenValidator {
	return c.MockValidator
}

func (c *MockClients) DDB() db.AWSDynamoDB {
	return c.MockDynamo
}

func (c *MockClients) Settings() appenv.Settings {
	return c.settings
}

func (c *MockClients) RootTenantID() uuid.UUID {
	return c.rootTenantID
}

func MockClientsOverride(t *testing.T) (*MockClients, func()) {
	cs := &MockClients{
		ctx:           context.Background(),
		cfg:           aws.Config{Region: "mock-region"},
		MockCogIdp:    mocks.NewAWSCognitoIdentityProvider(t),
		MockSES:       mocks.NewAWSSES(t),
		MockValidator: mocks.NewTokenValidator(t),
		MockDynamo:    mocks.NewAWSDynamoDB(t),
		settings:      appenv.MustSettings(),
		rootTenantID:  uuid.New(),
	}
	appcore.OverrideClient = cs
	return cs, func() {
		appcore.OverrideClient = nil
	}
}
