package authx_test

import (
	"testing"

	"github.com/apresdev/landlord/internal/db"

	"github.com/apresdev/landlord/internal/appcore"
	"github.com/apresdev/landlord/internal/appenv"
	"github.com/apresdev/landlord/internal/apptest"
	"github.com/apresdev/landlord/internal/authx"
	"github.com/apresdev/landlord/internal/multi"
	"github.com/stretchr/testify/suite"
)

type TenantsTestSuite struct {
	suite.Suite
	s appenv.Settings
}

func TestTenantsTestSuite(t *testing.T) {
	suite.Run(t, &TenantsTestSuite{s: appenv.MustTestSettings()})
}

func (s *TenantsTestSuite) SetupSuite() {
	if !apptest.InitEmulator(s.s) {
		s.T().Skip("AWS emulator not running")
	}
}

func (s *TenantsTestSuite) SetupTest() {
	apptest.ResetEmulator(s.s)
}

func (s *TenantsTestSuite) TestSetTenantStateOnRoot() {
	a := appcore.MustWithSettings(s.s)

	ten, err := db.GetRootTenant(a.Ctx(), a.DBCtx(), a.RootTenantID())
	s.Assert().NoError(err)

	v1, err := authx.SetTenantState(a.Ctx(), a.DBCtx(), ten, multi.TenantStateActive)
	s.Assert().Error(err)
	s.Assert().Nil(v1)

	v2, err := authx.SetTenantState(a.Ctx(), a.DBCtx(), ten, multi.TenantStateInactive)
	s.Assert().Error(err)
	s.Assert().Nil(v2)
}

func (s *TenantsTestSuite) TestSetTenantStateOnRegularTenant() {
	a := appcore.MustWithSettings(s.s)

	ten, err := authx.CreateTenant(
		a.Ctx(), a.DBCtx(), a.RootTenantID(), a.CogIdp(), "regular", "Regular Tenant")
	s.Assert().NoError(err)
	s.Assert().NotNil(ten)

	th, err := multi.ParseTenantHandle("regular")
	s.Assert().NoError(err)
	ten, err = db.ResolveTenant(a.Ctx(), a.DBCtx(), th)
	s.Assert().NoError(err)
	s.Assert().NotNil(ten)

	v1, err := authx.SetTenantState(a.Ctx(), a.DBCtx(), ten, multi.TenantStateActive)
	s.Assert().NoError(err)
	s.Assert().NotNil(v1)
	s.Assert().Equal(multi.TenantStateActive, v1.State)

	v2, err := authx.SetTenantState(a.Ctx(), a.DBCtx(), ten, multi.TenantStateInactive)
	s.Assert().NoError(err)
	s.Assert().NotNil(v2)
	s.Assert().Equal(multi.TenantStateInactive, v2.State)
}
