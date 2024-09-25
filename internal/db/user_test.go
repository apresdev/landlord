package db_test

import (
	"testing"

	"github.com/apresdev/landlord/internal/appcore"
	"github.com/apresdev/landlord/internal/appenv"
	"github.com/apresdev/landlord/internal/apptest"
	"github.com/apresdev/landlord/internal/db"
	"github.com/apresdev/landlord/internal/localstack"
	"github.com/google/uuid"

	"github.com/apresdev/landlord/internal/multi"
	"github.com/stretchr/testify/suite"
)

type UserTestSuite struct {
	suite.Suite
}

func TestUsersTestSuite(t *testing.T) {
	if !localstack.IsRunning() {
		t.Skip("localstack not running")
	}
	suite.Run(t, &UserTestSuite{})
}

func (s *UserTestSuite) SetupTest() {
	apptest.DangerousDeleteAllItems(appenv.MustSettings().DBTableName())
}

func (s *UserTestSuite) storeTestUser(a appcore.Clients) *multi.User {
	return s.storeTestUserWithName(a, "testuser@example.com")
}

func (s *UserTestSuite) storeTestUserWithName(a appcore.Clients, name string) *multi.User {
	u := &multi.User{
		TenantID: a.RootTenantID(),
		ID:       uuid.New(),
		Username: name,
		Role:     "admin",
		Status:   multi.UserStatusActive,
	}
	err := db.StoreUser(a.Ctx(), a.DDB(), u)
	s.NoError(err)
	return u
}

func (s *UserTestSuite) TestStoreUser() {
	a := appcore.Must()
	u1 := s.storeTestUser(a)
	u2, err := db.GetUserByName(a.Ctx(), a.DDB(), u1.Username)
	s.NoError(err)
	s.NotNil(u2)
	s.Equal(a.RootTenantID(), u2.TenantID)
	s.Equal(u1.ID, u2.ID)
	s.Equal(u1.Username, u2.Username)
	s.Equal(u1.Role, u2.Role)
}

func (s *UserTestSuite) TestDeleteUser() {
	a := appcore.Must()
	u1 := s.storeTestUser(a)
	u2, err := db.GetUserByName(a.Ctx(), a.DDB(), u1.Username)
	s.NoError(err)
	s.NotNil(u2)

	err = db.DeleteUser(a.Ctx(), a.DDB(), u2)
	s.NoError(err)

	u3, err := db.GetUserByName(a.Ctx(), a.DDB(), u1.Username)
	s.Error(err)
	s.Nil(u3)
}

func (s *UserTestSuite) TestGetUserByID() {
	a := appcore.Must()
	u1 := s.storeTestUser(a)
	u2, err := db.GetUserInTenantByID(a.Ctx(), a.DDB(), u1.TenantID, u1.ID)
	s.NoError(err)
	s.NotNil(u2)
	s.Equal(u1, u2)

	u3, err := db.GetUserInTenantByID(a.Ctx(), a.DDB(), u1.TenantID, uuid.New())
	s.Error(err)
	s.Nil(u3)
}

func (s *UserTestSuite) TestGetUsersForTenant() {
	a := appcore.Must()
	u1 := s.storeTestUserWithName(a, "testuser1@example.com")
	u2 := s.storeTestUserWithName(a, "testuser2@example.com")
	users1, err := db.GetUsersForTenant(a.Ctx(), a.DDB(), u1.TenantID)
	s.NoError(err)
	s.Len(users1, 2)
	s.Contains(users1, u1)
	s.Contains(users1, u2)

	t1 := &multi.Tenant{
		ID:   uuid.New(),
		Name: "other-tenant",
	}
	err = db.StoreTenant(a.Ctx(), a.DDB(), t1)
	s.NoError(err)
	u := &multi.User{
		TenantID: t1.ID,
		ID:       uuid.New(),
		Username: "testuser3@other.example.com",
		Role:     "admin",
		Status:   multi.UserStatusActive,
	}
	err = db.StoreUser(a.Ctx(), a.DDB(), u)
	s.NoError(err)

	users2, err := db.GetUsersForTenant(a.Ctx(), a.DDB(), u1.TenantID)
	s.NoError(err)
	s.Len(users2, 2)
	s.Contains(users2, u1)
	s.Contains(users2, u2)
}

func (s *UserTestSuite) TestGetUserAndTenant() {
	a := appcore.Must()
	u1 := s.storeTestUser(a)
	u2, t2, err := db.GetUserAndTenant(a.Ctx(), a.DDB(), u1.Username)
	s.NoError(err)
	s.NotNil(u2)
	s.NotNil(t2)
	s.Equal(u1, u2)
	s.Equal(a.RootTenantID(), t2.ID)
}

func (s *UserTestSuite) TestSetUserStatus() {
	a := appcore.Must()
	u1 := s.storeTestUser(a)
	u2, err := db.GetUserByName(a.Ctx(), a.DDB(), u1.Username)
	s.NoError(err)
	s.NotNil(u2)
	s.Equal(multi.UserStatusActive, u2.Status)

	u3, err := db.SetUserStatus(a.Ctx(), a.DDB(), u2, multi.UserStatusInactive)
	s.NoError(err)

	u4, err := db.GetUserByName(a.Ctx(), a.DDB(), u1.Username)
	s.NoError(err)
	s.NotNil(u4)
	s.Equal(multi.UserStatusInactive, u3.Status)

	u5, err := db.SetUserStatus(a.Ctx(), a.DDB(), u4, multi.UserStatusInactive)
	s.NoError(err)
	s.Equal(u4, u5)

	u6 := &multi.User{
		TenantID: a.RootTenantID(),
		ID:       uuid.New(),
		Username: "testuser6@example.com",
		Role:     "admin",
		Status:   multi.UserStatusInvited,
	}
	err = db.StoreUser(a.Ctx(), a.DDB(), u6)
	s.NoError(err)

	_, err = db.SetUserStatus(a.Ctx(), a.DDB(), u6, multi.UserStatusActive)
	s.Error(err)
	_, err = db.SetUserStatus(a.Ctx(), a.DDB(), u6, multi.UserStatusInactive)
	s.Error(err)
}
