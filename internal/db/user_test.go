package db_test

import (
	"testing"

	"github.com/apresdev/landlord/internal/appcore"
	"github.com/apresdev/landlord/internal/appenv"
	"github.com/apresdev/landlord/internal/apptest"
	"github.com/apresdev/landlord/internal/db"
	"github.com/google/uuid"

	"github.com/apresdev/landlord/internal/multi"
	"github.com/stretchr/testify/suite"
)

type UserTestSuite struct {
	suite.Suite
	s appenv.Settings
}

func TestUsersTestSuite(t *testing.T) {
	suite.Run(t, &UserTestSuite{s: appenv.MustTestSettings()})
}

func (s *UserTestSuite) SetupSuite() {
	if !apptest.InitLocalStack(s.s) {
		s.T().Skip("localstack not running")
	}
}

func (s *UserTestSuite) SetupTest() {
	apptest.ResetLocalStack(s.s)
}

func (s *UserTestSuite) storeTestUser(a appcore.Clients) *multi.User {
	return s.storeTestUserWithName(a, "testuser@example.com")
}

func (s *UserTestSuite) storeTestUserWithName(a appcore.Clients, name string) *multi.User {
	u := multi.NewUser(a.RootTenantID(), uuid.New(), name, "admin", multi.MFAPolicyNone)
	u.Status = multi.UserStatusActive
	err := db.StoreUser(a.Ctx(), a.DBCtx(), u)
	s.NoError(err)
	return u
}

func (s *UserTestSuite) TestStoreUser() {
	a := appcore.MustWithSettings(s.s)
	u1 := s.storeTestUser(a)
	u2, err := db.GetUserByName(a.Ctx(), a.DBCtx(), u1.Username)
	s.NoError(err)
	s.NotNil(u2)
	s.Equal(a.RootTenantID(), u2.TenantID)
	s.Equal(u1.ID, u2.ID)
	s.Equal(u1.CognitoID, u2.CognitoID)
	s.Equal(u1.Username, u2.Username)
	s.Equal(u1.Role, u2.Role)
}

func (s *UserTestSuite) TestDeleteUser() {
	a := appcore.MustWithSettings(s.s)
	u1 := s.storeTestUser(a)
	u2, err := db.GetUserByName(a.Ctx(), a.DBCtx(), u1.Username)
	s.NoError(err)
	s.NotNil(u2)

	err = db.DeleteUser(a.Ctx(), a.DBCtx(), u2)
	s.NoError(err)

	u3, err := db.GetUserByName(a.Ctx(), a.DBCtx(), u1.Username)
	s.Error(err)
	s.Nil(u3)
}

func (s *UserTestSuite) TestGetUserByID() {
	a := appcore.MustWithSettings(s.s)
	u1 := s.storeTestUser(a)
	u2, err := db.GetUserInTenantByID(a.Ctx(), a.DBCtx(), u1.TenantID, u1.ID)
	s.NoError(err)
	s.NotNil(u2)
	s.Equal(u1, u2)

	u3, err := db.GetUserInTenantByID(a.Ctx(), a.DBCtx(), u1.TenantID, uuid.New())
	s.Error(err)
	s.Nil(u3)
}

func (s *UserTestSuite) TestGetUsersForTenant() {
	a := appcore.MustWithSettings(s.s)
	u1 := s.storeTestUserWithName(a, "testuser1@example.com")
	u2 := s.storeTestUserWithName(a, "testuser2@example.com")
	users1, err := db.GetUsersForTenant(a.Ctx(), a.DBCtx(), u1.TenantID)
	s.NoError(err)
	s.Len(users1, 2)
	s.Contains(users1, u1)
	s.Contains(users1, u2)

	t1 := &multi.Tenant{
		ID:   uuid.New(),
		Name: "other-tenant",
	}
	err = db.StoreTenant(a.Ctx(), a.DBCtx(), t1)
	s.NoError(err)
	u := multi.NewUser(t1.ID, uuid.New(),
		"testuser3@other.example.com", "admin", multi.MFAPolicyRequired)
	u.Status = multi.UserStatusActive
	err = db.StoreUser(a.Ctx(), a.DBCtx(), u)
	s.NoError(err)

	users2, err := db.GetUsersForTenant(a.Ctx(), a.DBCtx(), u1.TenantID)
	s.NoError(err)
	s.Len(users2, 2)
	s.Contains(users2, u1)
	s.Contains(users2, u2)
}

func (s *UserTestSuite) TestGetUserAndTenant() {
	a := appcore.MustWithSettings(s.s)
	u1 := s.storeTestUser(a)
	u2, t2, err := db.GetUserAndTenant(a.Ctx(), a.DBCtx(), u1.Username)
	s.NoError(err)
	s.NotNil(u2)
	s.NotNil(t2)
	s.Equal(u1, u2)
	s.Equal(a.RootTenantID(), t2.ID)
}

func (s *UserTestSuite) TestSetUserStatus() {
	a := appcore.MustWithSettings(s.s)
	u1 := s.storeTestUser(a)
	u2, err := db.GetUserByName(a.Ctx(), a.DBCtx(), u1.Username)
	s.NoError(err)
	s.NotNil(u2)
	s.Equal(multi.UserStatusActive, u2.Status)

	u3, err := db.SetUserStatus(a.Ctx(), a.DBCtx(), u2, multi.UserStatusInactive)
	s.NoError(err)

	u4, err := db.GetUserByName(a.Ctx(), a.DBCtx(), u1.Username)
	s.NoError(err)
	s.NotNil(u4)
	s.Equal(multi.UserStatusInactive, u3.Status)

	u5, err := db.SetUserStatus(a.Ctx(), a.DBCtx(), u4, multi.UserStatusInactive)
	s.NoError(err)
	s.Equal(u4, u5)

	u6 := multi.NewUser(a.RootTenantID(), uuid.New(),
		"testuser6@example.com", "admin", multi.MFAPolicyNone)
	err = db.StoreUser(a.Ctx(), a.DBCtx(), u6)
	s.NoError(err)

	_, err = db.SetUserStatus(a.Ctx(), a.DBCtx(), u6, multi.UserStatusActive)
	s.Error(err)
	_, err = db.SetUserStatus(a.Ctx(), a.DBCtx(), u6, multi.UserStatusInactive)
	s.Error(err)
}

func (s *UserTestSuite) TestStoreUserError() {
	a := appcore.MustWithSettings(s.s)
	u := multi.NewUser(a.RootTenantID(), uuid.New(),
		"testuser7@other.example.com", "admin", multi.MFAPolicyRequired)
	u.Status = multi.UserStatusActive
	u.CognitoID = uuid.Nil
	err := db.StoreUser(a.Ctx(), a.DBCtx(), u)
	s.Error(err)
	s.Equal("cannot store user with nil Cognito ID", err.Error())
}
