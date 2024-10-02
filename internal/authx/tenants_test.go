package authx_test

import (
	"testing"

	"github.com/apresdev/landlord/internal/appcore"
	"github.com/apresdev/landlord/internal/appenv"
	"github.com/apresdev/landlord/internal/apptest"
	"github.com/apresdev/landlord/internal/localstack"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/apresdev/landlord/internal/multi"
	"github.com/stretchr/testify/suite"
)

type TenantsTestSuite struct {
	suite.Suite
}

func TestTenantsTestSuite(t *testing.T) {
	if !localstack.IsRunning() {
		t.Skip("localstack not running")
	}
	suite.Run(t, &TenantsTestSuite{})
}

func (s *TenantsTestSuite) SetupTest() {
	apptest.DangerousDeleteAllItems(appenv.MustSettings().DBTableName())
}

func (s *TenantsTestSuite) TestSetTenantStateOnRoot() {
	a := appcore.Must()

	th, err := multi.ParseTenantHandle("root")
	s.Assert().NoError(err)

	v1, err := authx.SetTenantState(a.Ctx(), a.DDB(), th, multi.TenantStateActive)
	s.Assert().Error(err)
	s.Assert().Nil(v1)

	v2, err := authx.SetTenantState(a.Ctx(), a.DDB(), th, multi.TenantStateInactive)
	s.Assert().Error(err)
	s.Assert().Empty(v2)
}

func (s *TenantsTestSuite) TestSetTenantStateOnRegularTenant() {
	a := appcore.Must()

	ten, err := authx.CreateTenant(
		a.Ctx(), a.DDB(), a.RootTenantID(), a.CogIdp(), "regular", "Regular Tenant")
	s.Assert().NoError(err)
	s.Assert().NotNil(ten)

	th, err := multi.ParseTenantHandle("regular")
	s.Assert().NoError(err)

	v1, err := authx.SetTenantState(a.Ctx(), a.DDB(), th, multi.TenantStateActive)
	s.Assert().NoError(err)
	s.Assert().NotNil(v1)
	s.Assert().Equal(multi.TenantStateActive, v1.State)

	v2, err := authx.SetTenantState(a.Ctx(), a.DDB(), th, multi.TenantStateInactive)
	s.Assert().NoError(err)
	s.Assert().NotNil(v2)
	s.Assert().Equal(multi.TenantStateInactive, v2.State)
}
