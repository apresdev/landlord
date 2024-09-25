package authx_test

import (
	"context"
	"errors"
	"testing"

	"github.com/apresdev/landlord/internal/appcore"
	"github.com/apresdev/landlord/internal/appenv"
	"github.com/apresdev/landlord/internal/db"
	"github.com/apresdev/landlord/internal/localstack"
	"github.com/google/uuid"

	"github.com/apresdev/landlord/internal/apptest"
	"github.com/stretchr/testify/mock"

	"github.com/apresdev/landlord/internal/multi"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/apresdev/landlord/internal/mocks"
	"github.com/stretchr/testify/suite"
)

type InviteTestSuite struct {
	suite.Suite
}

func TestInviteTestSuite(t *testing.T) {
	if !localstack.IsRunning() {
		t.Skip("localstack not running")
	}
	suite.Run(t, &InviteTestSuite{})
}

func (s *InviteTestSuite) SetupTest() {
	apptest.DangerousDeleteAllItems(appenv.MustSettings().DBTableName())
	apptest.DangerousRecreateUserPool()
}

func (s *InviteTestSuite) TestBadEmail() {
	a := appcore.Must()
	rootTenant, err := db.GetRootTenant(a.Ctx(), a.DDB(), a.RootTenantID())
	s.Require().NoError(err)
	err = authx.SendInvite(
		a.Ctx(),
		a.DDB(),
		a.CogIdp(),
		rootTenant,
		"not-an-email-address",
		"guest",
		multi.MFAPolicyNone)
	s.Error(err)
	s.Equal(err.Error(), "invalid email")
}

func (s *InviteTestSuite) TestInviteDBFailure() {
	ctx := context.Background()
	ddb := mocks.NewAWSDynamoDB(s.T())
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	ddb.EXPECT().Query(mock.Anything, mock.Anything).Return(nil, errors.New("db error"))
	tenant := multi.NewTenant()
	err := authx.SendInvite(
		ctx, ddb, cogIdp, tenant, "foo@example.com", "guest", multi.MFAPolicyNone)
	s.Error(err)
	s.Contains(err.Error(), "db error")
}

func (s *InviteTestSuite) TestInviteInactiveUser() {
	a := appcore.Must()

	ten, err := authx.CreateTenant(
		a.Ctx(), a.DDB(), a.RootTenantID(), a.CogIdp(), "foo", "Foo")
	s.Require().NoError(err)
	s.Require().NotNil(ten)

	err = db.StoreUser(a.Ctx(), a.DDB(), &multi.User{
		ID:       uuid.New(),
		TenantID: ten.ID,
		Username: "foo@example.com",
		Role:     multi.RoleGuest,
		Status:   multi.UserStatusInactive,
	})
	s.Require().NoError(err)

	err = authx.SendInvite(
		a.Ctx(),
		a.DDB(),
		a.CogIdp(),
		ten,
		"foo@example.com",
		"guest",
		multi.MFAPolicyNone)
	s.Error(err)
	s.Equal(err.Error(), "cannot send invite, user 'foo@example.com' is inactive")
}

func (s *InviteTestSuite) TestInviteNewUser() {
	a := appcore.Must()

	ten, err := authx.CreateTenant(
		a.Ctx(), a.DDB(), a.RootTenantID(), a.CogIdp(), "foo", "Foo")
	s.Require().NoError(err)
	s.Require().NotNil(ten)

	err = authx.SendInvite(
		a.Ctx(),
		a.DDB(),
		a.CogIdp(),
		ten,
		"foo@example.com",
		"guest",
		multi.MFAPolicyNone)
	s.NoError(err)
}
