package rest

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/apresdev/landlord/internal/perms"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/apresdev/landlord/internal/multi"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/suite"
)

type HandleRetrieveTenantTestSuite struct {
	suite.Suite
	ginMode string
}

func TestHandleRetrieveTenantTestSuite(t *testing.T) {
	suite.Run(t, &HandleRetrieveTenantTestSuite{})
}

func (s *HandleRetrieveTenantTestSuite) SetupTest() {
	s.ginMode = gin.Mode()
	gin.SetMode(gin.TestMode)
}

func (s *HandleRetrieveTenantTestSuite) TearDownTest() {
	gin.SetMode(s.ginMode)
}

func newGetTenantByIDTestSetup(t *testing.T, ten *multi.Tenant, user *multi.User) *ginTest {
	r := httptest.NewRequest("GET", "/tenant", nil)
	gt := newGinTest(t).withRequest(r)
	if ten != nil {
		claims := make(jwt.MapClaims)
		claims[authx.CustomTenantClaim] = ten.ID.String()
		if user != nil {
			claims[authx.CognitoGroupsClaim] = []string{ten.GroupName(user.Role)}
			claims[authx.EmailClaim] = user.Username
			claims[authx.SubClaim] = user.CognitoID.String()
			claims[authx.CustomUserClaim] = user.ID.String()
		}
		tok := &jwt.Token{Claims: claims}
		gt.c.Set(perms.TokenKey, tok)
	}
	return gt
}

func (s *HandleRetrieveTenantTestSuite) TestNotAuthorized() {
	app := NewMockApp()
	gt1 := newGetTenantByIDTestSetup(s.T(), nil, nil)
	err := handleRetrieveTenant(app, gt1.c)
	s.Error(err)
	s.Equal("no token", err.Error())
}

func (s *HandleRetrieveTenantTestSuite) TestBadRequests() {
	app := NewMockApp()

	ten := multi.NewTenant("ten1")
	user := newTestUser(ten)

	gt1 := newGetTenantByIDTestSetup(s.T(), ten, user)
	err := handleRetrieveTenant(app, gt1.c)
	s.Error(err)
	s.Equal("no tenantID provided", err.Error())

	gt1.c.Params = gin.Params{{Key: "tenantID", Value: ""}}
	err = handleRetrieveTenant(app, gt1.c)
	s.Error(err)
	s.Equal("no tenantID provided", err.Error())

	gt1.c.Params = gin.Params{{Key: "tenantID", Value: "123456789012345678901234567890123"}}
	err = handleRetrieveTenant(app, gt1.c)
	s.Error(err)
	s.Equal("invalid tenantID", err.Error())
}

func (s *HandleRetrieveTenantTestSuite) TestNotFound() {
	app := NewMockApp()
	ten := multi.NewTenant("ten1")
	user := newTestUser(ten)
	gt1 := newGetTenantByIDTestSetup(s.T(), ten, user)
	gt1.c.Params = gin.Params{{Key: "tenantID", Value: uuid.New().String()}}
	app.ddb.EXPECT().GetItem(mock.Anything, mock.Anything).Return(nil, errors.New("not found"))
	err := handleRetrieveTenant(app, gt1.c)
	s.Error(err)
}

func (s *HandleRetrieveTenantTestSuite) TestSuccess() {
	app := NewMockApp()
	ten := multi.NewTenant("ten1")
	user := newTestUser(ten)
	gt1 := newGetTenantByIDTestSetup(s.T(), ten, user)
	gt1.c.Params = gin.Params{{Key: "tenantID", Value: ten.ID.String()}}
	withSetupGetTenantByID2(app, ten)
	err := handleRetrieveTenant(app, gt1.c)
	s.NoError(err)
}

func (s *HandleRetrieveTenantTestSuite) TestNotAuthorizedForTenant() {
	// user1 is in ten1, but requesting ten2 -- not authorized
	app := NewMockApp()
	ten1 := multi.NewTenant("ten1")
	ten2 := multi.NewTenant("ten2")
	user1 := newTestUser(ten1)
	gt1 := newGetTenantByIDTestSetup(s.T(), ten1, user1)

	gt1.c.Params = gin.Params{{Key: "tenantID", Value: ten2.ID.String()}}
	withSetupGetTenantByID2(app, ten2)
	err := handleRetrieveTenant(app, gt1.c)
	s.Error(err)
	s.True(apperr.IsUnauthorized(err))
	s.Equal("not authorized", err.Error())
}

func (s *HandleRetrieveTenantTestSuite) TestAuthorizedViaRoot() {
	// user1 is in root, requesting ten1 -- authorized
	app := NewMockApp()
	ten1 := multi.NewTenant("ten1")
	rt := multi.NewTenant(multi.RootTenantName)
	rt.ID = app.RootTenantID()
	user1 := newTestUser(rt)
	gt1 := newGetTenantByIDTestSetup(s.T(), ten1, user1)
	gt1.c.Params = gin.Params{{Key: "tenantID", Value: ten1.ID.String()}}
	withSetupGetTenantByID2(app, ten1)
	err := handleRetrieveTenant(app, gt1.c)
	s.NoError(err)
}
