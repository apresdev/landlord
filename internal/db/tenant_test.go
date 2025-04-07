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

type TenantTestSuite struct {
	suite.Suite
	s appenv.Settings
}

func TestTenantTestSuite(t *testing.T) {
	suite.Run(t, &TenantTestSuite{s: appenv.MustTestSettings()})
}

func (s *TenantTestSuite) SetupSuite() {
	if !apptest.InitLocalStack(s.s) {
		s.T().Skip("localstack not running")
	}
}

func (s *TenantTestSuite) SetupTest() {
	apptest.ResetLocalStack(s.s)
}

func (s *TenantTestSuite) storeTestTenant(a appcore.Clients) *multi.Tenant {
	return s.storeTestTenantWithName(a, "test-tenant")
}

func (s *TenantTestSuite) storeTestTenantWithName(
	a appcore.Clients, name string) *multi.Tenant {

	t := &multi.Tenant{
		ID:   uuid.New(),
		Name: name,
	}
	err := db.StoreTenant(a.Ctx(), a.DBCtx(), t)
	s.NoError(err)
	return t
}

func (s *TenantTestSuite) TestStoreTenant() {
	a := appcore.MustWithSettings(s.s)
	t1 := s.storeTestTenant(a)
	t2, err := db.GetTenantByID(a.Ctx(), a.DBCtx(), t1.ID)
	s.NoError(err)
	s.NotNil(t2)
	s.Equal(t1, t2)
}

func (s *TenantTestSuite) TestGetTenantByID() {
	a := appcore.MustWithSettings(s.s)
	t1 := s.storeTestTenant(a)
	t2, err := db.GetTenantByID(a.Ctx(), a.DBCtx(), t1.ID)
	s.NoError(err)
	s.NotNil(t2)
	s.Equal(t1, t2)

	t3, err := db.GetTenantByID(a.Ctx(), a.DBCtx(), uuid.New())
	s.Error(err)
	s.Nil(t3)

	t4, err := db.GetRootTenant(a.Ctx(), a.DBCtx(), a.RootTenantID())
	s.NoError(err)
	s.NotNil(t4)
	s.Equal(a.RootTenantID(), t4.ID)
	s.Equal("root", t4.Name)
}

func (s *TenantTestSuite) TestGetTenantByName() {
	a := appcore.MustWithSettings(s.s)
	t1 := s.storeTestTenant(a)
	t2, err := db.GetTenantByName(a.Ctx(), a.DBCtx(), t1.Name)
	s.NoError(err)
	s.NotNil(t2)
	s.Equal(t1, t2)

	t3, err := db.GetTenantByName(a.Ctx(), a.DBCtx(), "non-existent")
	s.Error(err)
	s.Nil(t3)
}

func (s *TenantTestSuite) TestGetAllTenants() {
	a := appcore.MustWithSettings(s.s)
	tc := s.storeTestTenantWithName(a, "c-test-tenant1")
	tb := s.storeTestTenantWithName(a, "b-test-tenant2")
	ta := s.storeTestTenantWithName(a, "a-test-tenant3")

	ts, err := db.GetAllTenants(a.Ctx(), a.DBCtx())
	s.NoError(err)
	// Check that GetAllTenants sorted for us, root is put first
	s.Len(ts, 4)
	s.Equal("root", ts[0].Name)
	s.Equal(ta.Name, ts[1].Name)
	s.Equal(tb.Name, ts[2].Name)
	s.Equal(tc.Name, ts[3].Name)
}
