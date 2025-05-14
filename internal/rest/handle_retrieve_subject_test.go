package rest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/apresdev/landlord/internal/appcore"
	"github.com/apresdev/landlord/internal/appenv"
	"github.com/apresdev/landlord/internal/apptest"
	"github.com/apresdev/landlord/internal/authx"
	"github.com/apresdev/landlord/internal/db"
	"github.com/apresdev/landlord/internal/multi"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/google/uuid"
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
	if !apptest.InitLocalStack(s.s) {
		s.T().Skip("localstack not running")
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

func (s *HandleRetrieveSubjectTestSuite) TestHappyPath() {
	ctx := context.Background()
	emailAddress := uuid.New().String() + "@apres.dev"
	rootTenant, err := db.GetRootTenant(ctx, s.app.DBCtx(), s.app.RootTenantID())
	s.Require().NoError(err)
	err = authx.SendInvite(
		ctx,
		s.app.DBCtx(),
		s.app.CogIdp(),
		rootTenant,
		emailAddress,
		uuid.Nil,
		"admin",
		multi.MFAPolicyNone)
	s.NoError(err)
	pwd := generateSortOfRandomPassword(12)
	_, err = s.app.CogIdp().AdminSetUserPassword(
		ctx,
		&cognitoidentityprovider.AdminSetUserPasswordInput{
			Password:   aws.String(pwd),
			UserPoolId: aws.String(rootTenant.UserPoolID),
			Username:   aws.String(emailAddress),
			Permanent:  true,
		})
	s.NoError(err)

	upc, err := authx.GetUserPoolClient(ctx, s.app.CogIdp(), rootTenant.UserPoolID)
	s.NoError(err)

	params := make(map[string]string)
	params["USERNAME"] = emailAddress
	params["PASSWORD"] = pwd
	secretHash, err := authx.CreateSecretHash(
		emailAddress, aws.ToString(upc.ClientId), aws.ToString(upc.ClientSecret))
	s.NoError(err)
	params["SECRET_HASH"] = secretHash

	o, err := s.app.CogIdp().InitiateAuth(
		ctx,
		&cognitoidentityprovider.InitiateAuthInput{
			AuthFlow:       "USER_PASSWORD_AUTH",
			ClientId:       upc.ClientId,
			AuthParameters: params,
		})
	s.NoError(err)
	accessToken := aws.ToString(o.AuthenticationResult.AccessToken)

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/ll-auth/api/subject", nil)
	s.NoError(err)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	s.g.ServeHTTP(w, req)
	assert.Equal(s.T(), http.StatusOK, w.Code)
	bodyBytes, err := io.ReadAll(w.Body)
	s.NoError(err)
	resp := &SubjectResponse{}
	err = json.Unmarshal(bodyBytes, resp)
	s.NoError(err)
	assert.Equal(s.T(), emailAddress, resp.Username)
	assert.Equal(s.T(), rootTenant.ID.String(), resp.TenantID)
	assert.Equal(s.T(), rootTenant.Name, resp.TenantName)
	assert.Equal(s.T(), "admin", resp.Role)
	assert.Equal(s.T(), UserStatus("invited"), resp.Status)
	assert.NotEqual(s.T(), "", resp.CognitoID)
	assert.NotEqual(s.T(), "", resp.UserID)
}
