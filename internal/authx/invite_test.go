package authx_test

import (
	"context"
	"errors"
	"testing"

	"github.com/apresdev/landlord/internal/appcore"
	"github.com/apresdev/landlord/internal/appenv"
	"github.com/apresdev/landlord/internal/db"
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
	s appenv.Settings
}

func TestInviteTestSuite(t *testing.T) {
	suite.Run(t, &InviteTestSuite{s: appenv.MustTestSettings()})
}

func (s *InviteTestSuite) SetupSuite() {
	if !apptest.InitLocalStack(s.s) {
		s.T().Skip("localstack not running")
	}
}

func (s *InviteTestSuite) SetupTest() {
	apptest.ResetLocalStack(s.s)
}

func (s *InviteTestSuite) TestBadEmail() {
	a := appcore.MustWithSettings(s.s)
	rootTenant, err := db.GetRootTenant(a.Ctx(), a.DBCtx(), a.RootTenantID())
	s.Require().NoError(err)
	err = authx.SendInvite(
		a.Ctx(),
		a.DBCtx(),
		a.CogIdp(),
		rootTenant,
		"not-an-email-address",
		uuid.Nil,
		"guest",
		multi.MFAPolicyNone)
	s.Error(err)
	s.Equal(err.Error(), "invalid email")
}

func (s *InviteTestSuite) TestInviteDBFailure() {
	ctx := context.Background()
	ddb := mocks.NewAWSDynamoDB(s.T())
	dbc := db.NewContext("test-table-name", ddb)
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	ddb.EXPECT().Query(mock.Anything, mock.Anything).Return(nil, errors.New("db error"))
	tenant := multi.NewTenant("ten1")
	err := authx.SendInvite(
		ctx,
		dbc,
		cogIdp,
		tenant,
		"foo@example.com",
		uuid.Nil,
		"guest",
		multi.MFAPolicyNone)
	s.Error(err)
	s.Contains(err.Error(), "db error")
}

func (s *InviteTestSuite) TestInviteInactiveUser() {
	a := appcore.MustWithSettings(s.s)

	ten, err := authx.CreateTenant(
		a.Ctx(), a.DBCtx(), a.RootTenantID(), a.CogIdp(), "foo", "Foo")
	s.Require().NoError(err)
	s.Require().NotNil(ten)

	u := multi.NewUser(ten.ID, uuid.New(), "foo@example.com", "guest", multi.MFAPolicyNone)
	u.Status = multi.UserStatusInactive

	err = db.StoreUser(a.Ctx(), a.DBCtx(), u)
	s.Require().NoError(err)

	err = authx.SendInvite(
		a.Ctx(),
		a.DBCtx(),
		a.CogIdp(),
		ten,
		"foo@example.com",
		uuid.Nil,
		"guest",
		multi.MFAPolicyNone)
	s.Error(err)
	s.Equal(err.Error(), "cannot send invite, user 'foo@example.com' is inactive")
}

func (s *InviteTestSuite) TestInviteNewUser() {
	a := appcore.MustWithSettings(s.s)

	ten, err := authx.CreateTenant(
		a.Ctx(), a.DBCtx(), a.RootTenantID(), a.CogIdp(), "foo", "Foo")
	s.Require().NoError(err)
	s.Require().NotNil(ten)

	err = authx.SendInvite(
		a.Ctx(),
		a.DBCtx(),
		a.CogIdp(),
		ten,
		"foo@example.com",
		uuid.Nil,
		"guest",
		multi.MFAPolicyNone)
	s.NoError(err)
}

func (s *InviteTestSuite) TestInviteOverrideUserID() {
	a := appcore.MustWithSettings(s.s)

	ten, err := authx.CreateTenant(
		a.Ctx(), a.DBCtx(), a.RootTenantID(), a.CogIdp(), "foo", "Foo")
	s.Require().NoError(err)
	s.Require().NotNil(ten)

	overrideUserID := uuid.New()

	err = authx.SendInvite(
		a.Ctx(),
		a.DBCtx(),
		a.CogIdp(),
		ten,
		"foo@example.com",
		overrideUserID,
		"guest",
		multi.MFAPolicyNone)
	s.NoError(err)

	u1, err := db.GetUserByName(a.Ctx(), a.DBCtx(), "foo@example.com")
	s.NoError(err)
	s.Equal(overrideUserID, u1.ID)
}
