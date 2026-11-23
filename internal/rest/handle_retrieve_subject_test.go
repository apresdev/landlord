package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/apresdev/landlord/internal/appcore"
	"github.com/apresdev/landlord/internal/appenv"
	"github.com/apresdev/landlord/internal/apptest"
	"github.com/apresdev/landlord/internal/db"
	"github.com/stretchr/testify/assert"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
)

type HandleRetrieveSubjectTestSuite struct {
	suite.Suite
	ginMode string
	s       appenv.Settings
	g       *gin.Engine
	app     App
}

func TestHandleRetrieveSubjectTestSuite(t *testing.T) {
	suite.Run(t, &HandleRetrieveSubjectTestSuite{s: appenv.MustTestSettings()})
}

func (s *HandleRetrieveSubjectTestSuite) SetupSuite() {
	if !apptest.InitEmulator(s.s) {
		s.T().Skip("AWS emulator not running")
	}
}

func (s *HandleRetrieveSubjectTestSuite) SetupTest() {
	s.ginMode = gin.Mode()
	gin.SetMode(gin.TestMode)
	s.g = gin.New()
	s.app = appcore.MustWithSettings(s.s)
	AddRoutes(s.app, s.g)
}

func (s *HandleRetrieveSubjectTestSuite) TearDownTest() {
	gin.SetMode(s.ginMode)
}

func (s *HandleRetrieveSubjectTestSuite) TestNotARealToken() {
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/ll-auth/api/subject", nil)
	s.NoError(err)
	req.Header.Set("Authorization", "Bearer Not-A-Real-Token")
	s.g.ServeHTTP(w, req)
	assert.Equal(s.T(), http.StatusUnauthorized, w.Code)
}

func retrieveSubject(t *testing.T, g *gin.Engine, accessToken string) *SubjectResponse {
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/ll-auth/api/subject", nil)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	g.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	bodyBytes, err := io.ReadAll(w.Body)
	assert.NoError(t, err)
	resp := &SubjectResponse{}
	err = json.Unmarshal(bodyBytes, resp)
	assert.NoError(t, err)
	return resp
}

func (s *HandleRetrieveSubjectTestSuite) TestHappyPath() {
	invited := inviteUserAndAuthenticate(s.T(), s.app)
	subject := retrieveSubject(s.T(), s.g, invited.accessToken)

	ctx := context.Background()
	rootTenant, err := db.GetRootTenant(ctx, s.app.DBCtx(), s.app.RootTenantID())
	assert.NoError(s.T(), err)
	user, err := db.GetUserByName(ctx, s.app.DBCtx(), invited.emailAddress)
	assert.NoError(s.T(), err)

	assert.Equal(s.T(), invited.emailAddress, subject.Username)
	assert.Equal(s.T(), rootTenant.ID.String(), subject.TenantID)
	assert.Equal(s.T(), rootTenant.Name, subject.TenantName)
	assert.Equal(s.T(), "admin", subject.Role)
	assert.Equal(s.T(), UserStatus("invited"), subject.Status)
	assert.Equal(s.T(), user.CognitoID.String(), subject.CognitoID)
	assert.Equal(s.T(), user.ID.String(), subject.UserID)
	assert.Equal(s.T(), "", subject.Impersonating.TenantID)
}

func (s *HandleRetrieveSubjectTestSuite) TestImpersonate() {
	invited1 := inviteUserAndAuthenticate(s.T(), s.app)
	invited2 := inviteUserAndAuthenticate(s.T(), s.app)

	user2, err := db.GetUserByName(context.Background(), s.app.DBCtx(), invited2.emailAddress)
	assert.NoError(s.T(), err)

	subject1 := retrieveSubject(s.T(), s.g, invited1.accessToken)
	assert.Equal(s.T(), invited1.emailAddress, subject1.Username)
	assert.Equal(s.T(), "", subject1.Impersonating.TenantID)

	w := httptest.NewRecorder()
	reqBody := ImpersonateUserRequest{
		Tenant:       user2.TenantID.String(),
		Username:     user2.Username,
		RefreshToken: invited1.refreshToken,
	}
	reqBodyBytes, err := json.Marshal(reqBody)
	assert.NoError(s.T(), err)
	impersonateReq, err := http.NewRequest(
		"POST", "/ll-auth/api/impersonate-user", bytes.NewBuffer(reqBodyBytes))
	assert.NoError(s.T(), err)
	impersonateReq.Header.Set("Authorization", "Bearer "+invited1.accessToken)
	s.g.ServeHTTP(w, impersonateReq)
	assert.Equal(s.T(), http.StatusOK, w.Code)
	resp := ImpersonateResponse{}
	bodyBytes, err := io.ReadAll(w.Body)
	assert.NoError(s.T(), err)
	err = json.Unmarshal(bodyBytes, &resp)
	assert.NoError(s.T(), err)

	subject1a := retrieveSubject(s.T(), s.g, resp.AccessToken)
	assert.Equal(s.T(), invited1.emailAddress, subject1a.Username)
	assert.Equal(s.T(), user2.TenantID.String(), subject1a.Impersonating.TenantID)
}
