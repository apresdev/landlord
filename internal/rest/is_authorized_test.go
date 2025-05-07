package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"

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

func (s *IsAuthorizedTestSuite) TestHappyPath() {
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
