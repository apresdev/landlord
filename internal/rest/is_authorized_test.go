package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/apresdev/landlord/internal/apperr"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/mock"

	"github.com/apresdev/landlord/internal/appenv"
	"github.com/gin-gonic/gin"

	"github.com/stretchr/testify/suite"
)

type IsAuthorizedTestSuite struct {
	suite.Suite
	ginMode string
	s       appenv.Settings
}

func TestIsAuthorizedTestSuite(t *testing.T) {
	suite.Run(t, &IsAuthorizedTestSuite{s: appenv.MustTestSettings()})
}

func (s *IsAuthorizedTestSuite) SetupTest() {
	s.ginMode = gin.Mode()
	gin.SetMode(gin.TestMode)
}

func (s *IsAuthorizedTestSuite) TestBadRequests() {
	a := NewMockApp()
	f := isAuthorized(a)

	reqWithNoAuthHeader := httptest.NewRequest("GET", "/example", nil)
	reqWithEmptyAuthHeader := httptest.NewRequest("GET", "/example", nil)
	reqWithEmptyAuthHeader.Header.Set("Authorization", "")
	reqWithBadBearerHeader := httptest.NewRequest("GET", "/example", nil)
	reqWithBadBearerHeader.Header.Set("Authorization", "Bearer ")
	reqWithBasicHeader := httptest.NewRequest("GET", "/example", nil)
	reqWithBasicHeader.Header.Set("Authorization", "Basic dXNlcjpwYXNzd29yZA==")

	testReqsExpect401 := []*http.Request{
		reqWithNoAuthHeader,
		reqWithEmptyAuthHeader,
		reqWithBadBearerHeader,
		reqWithBasicHeader,
	}

	for _, req := range testReqsExpect401 {
		gt := newGinTest(s.T()).withRequest(req)
		f(gt.c)
		s.Equal(401, gt.c.Writer.Status())
	}
}

func (s *IsAuthorizedTestSuite) TestBadToken() {
	a := NewMockApp()
	withSetupRootTenant(a)
	f := isAuthorized(a)

	reqWithBadToken := httptest.NewRequest("GET", "/example", nil)
	reqWithBadToken.Header.Set("Authorization", "Bearer badtoken")
	a.validator.On("ValidateToken", mock.Anything, mock.Anything, mock.Anything, "badtoken").
		Return(nil, apperr.NewInternal("foo"))
	gt := newGinTest(s.T()).withRequest(reqWithBadToken)
	f(gt.c)
	s.Equal(401, gt.c.Writer.Status())
}

func (s *IsAuthorizedTestSuite) TestGoodToken() {
	a := NewMockApp()
	withSetupRootTenant(a)
	f := isAuthorized(a)

	reqWithBadToken := httptest.NewRequest("GET", "/example", nil)
	reqWithBadToken.Header.Set("Authorization", "Bearer goodtoken   ")
	a.validator.On("ValidateToken", mock.Anything, mock.Anything, mock.Anything, "goodtoken").
		Return(&jwt.Token{}, nil)
	gt := newGinTest(s.T()).withRequest(reqWithBadToken)
	f(gt.c)
	s.Equal(200, gt.c.Writer.Status())
}
