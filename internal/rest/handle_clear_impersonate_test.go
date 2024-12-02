package rest

import (
	"bytes"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/apresdev/landlord/internal/perms"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/stretchr/testify/suite"
)

type HandleClearImpersonationTestSuite struct {
	suite.Suite
	ginMode string
}

func TestHandleClearImpersonationTestSuite(t *testing.T) {
	suite.Run(t, &HandleClearImpersonationTestSuite{})
}

func (s *HandleClearImpersonationTestSuite) SetupTest() {
	s.ginMode = gin.Mode()
	gin.SetMode(gin.TestMode)
}

func (s *HandleClearImpersonationTestSuite) TearDownTest() {
	gin.SetMode(s.ginMode)
}

func withDefaultClearImpersonationRequest(gt *ginTest) *ginTest {
	return gt.withRequest(
		httptest.NewRequest(
			"POST",
			"/clear-impersonate",
			bytes.NewBuffer(MustJSONMarshal(
				ClearImpersonationRequest{RefreshToken: "refresh_token"}))))
}

func (s *HandleClearImpersonationTestSuite) TestBadRequests() {
	app := NewMockApp()
	gt1 := newGinTest(s.T()).withRequest(
		httptest.NewRequest("POST", "/clear-impersonate", nil))
	err := handleClearImpersonation(app, gt1.c)
	s.Error(err)

	gt2 := newGinTest(s.T()).withRequest(
		httptest.NewRequest(
			"POST", "/clear-impersonate", bytes.NewBuffer([]byte("{}"))))
	err = handleClearImpersonation(app, gt2.c)
	s.Error(err)

	gt3 := withDefaultClearImpersonationRequest(newGinTest(s.T()))
	err = handleClearImpersonation(app, gt3.c)
	s.Error(err)

	gt4 := withDefaultClearImpersonationRequest(newGinTest(s.T()))
	gt4.c.Set(perms.TokenKey, "not-the-right-type")
	err = handleClearImpersonation(app, gt4.c)
	s.Error(err)
}

func (s *HandleClearImpersonationTestSuite) TestBadTokens() {
	app := NewMockApp()
	gt1 := withDefaultClearImpersonationRequest(newGinTest(s.T()))
	gt1.c.Set(perms.TokenKey, &jwt.Token{})
	err := handleClearImpersonation(app, gt1.c)
	s.Error(err)

	gt2 := withDefaultClearImpersonationRequest(newGinTest(s.T()))
	claims := make(jwt.MapClaims)
	claims[authx.CognitoGroupsClaim] = []string{"group1", "group2"}
	tok := &jwt.Token{Claims: claims}
	gt2.c.Set(perms.TokenKey, tok)
	err = handleClearImpersonation(app, gt2.c)
	s.Error(err)
}

func (s *HandleClearImpersonationTestSuite) TestNoRootTenant() {
	app := NewMockApp()
	gt1 := withDefaultClearImpersonationRequest(newGinTest(s.T()))
	claims := make(jwt.MapClaims)
	tenantUUID := uuid.New().String()
	claims[authx.CognitoGroupsClaim] = []string{
		fmt.Sprintf("tenant-%s-guest", tenantUUID)}
	tok := &jwt.Token{Claims: claims}
	gt1.c.Set(perms.TokenKey, tok)
	app.ddb.EXPECT().GetItem(mock.Anything, mock.Anything).Return(
		nil, fmt.Errorf("error"))
	err := handleClearImpersonation(app, gt1.c)
	s.Error(err)
}

func (s *HandleClearImpersonationTestSuite) TestNotAuthorizedForRoot() {
	app := NewMockApp()
	gt1 := withDefaultClearImpersonationRequest(newGinTest(s.T()))
	claims := make(jwt.MapClaims)
	tenantUUID := uuid.New().String()
	claims[authx.CognitoGroupsClaim] = []string{
		fmt.Sprintf("tenant-%s-guest", tenantUUID)}
	tok := &jwt.Token{Claims: claims}
	gt1.c.Set(perms.TokenKey, tok)
	_ = withSetupRootTenant(app)
	err := handleClearImpersonation(app, gt1.c)
	s.Error(err)
}

func (s *HandleClearImpersonationTestSuite) TestNoEmailClaim() {
	app := NewMockApp()
	gt1 := withDefaultClearImpersonationRequest(newGinTest(s.T()))
	rt := withSetupRootTenant(app)
	claims := make(jwt.MapClaims)
	claims[authx.CognitoGroupsClaim] = []string{
		fmt.Sprintf("tenant-%s-guest", rt.ID.String())}
	claims[authx.CustomTenantClaim] = rt.ID.String()
	tok := &jwt.Token{Claims: claims}
	gt1.c.Set(perms.TokenKey, tok)
	err := handleClearImpersonation(app, gt1.c)
	s.Error(err)
}

func (s *HandleClearImpersonationTestSuite) TestFailedToUpdateUserAttributes() {
	app := NewMockApp()
	gt1 := withDefaultClearImpersonationRequest(newGinTest(s.T()))
	rt := withSetupRootTenant(app)
	u := newTestUser(rt)
	withSetupQueryUser2(app, u)
	withSetupJWT(gt1.c, rt, u)
	app.cogIdp.EXPECT().AdminUpdateUserAttributes(mock.Anything, mock.Anything).Return(
		nil, errors.New("could not update user attributes"))
	err := handleClearImpersonation(app, gt1.c)
	s.Error(err)
}

func (s *HandleClearImpersonationTestSuite) TestNoSubClaim() {
	app := NewMockApp()
	gt1 := withDefaultClearImpersonationRequest(newGinTest(s.T()))
	rt := withSetupRootTenant(app)
	u1 := newTestUser(rt)
	withSetupQueryUser2(app, u1)
	claims := make(jwt.MapClaims)
	claims[authx.CustomTenantClaim] = rt.ID.String()
	claims[authx.CognitoGroupsClaim] = []string{rt.GroupName(u1.Role)}
	claims[authx.EmailClaim] = u1.Username
	// Leave this claim out for the test
	// claims[authx.SubClaim] = u1.ID.String()
	tok := &jwt.Token{Claims: claims}
	gt1.c.Set(perms.TokenKey, tok)

	app.cogIdp.EXPECT().AdminUpdateUserAttributes(mock.Anything, mock.Anything).Return(
		nil, nil)
	app.cogIdp.EXPECT().ListUserPoolClients(gt1.c.Request.Context(), mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolClientsOutput{
			UserPoolClients: []types.UserPoolClientDescription{
				{ClientId: aws.String("test-client-id1")},
			}}, nil).Once()
	app.cogIdp.EXPECT().DescribeUserPoolClient(gt1.c.Request.Context(), mock.Anything).Return(
		&cognitoidentityprovider.DescribeUserPoolClientOutput{
			UserPoolClient: &types.UserPoolClientType{
				ClientId:     aws.String("test-client-id1"),
				ClientSecret: aws.String("test-client-secret")}}, nil).Once()
	err := handleClearImpersonation(app, gt1.c)
	s.Error(err)
}

func (s *HandleClearImpersonationTestSuite) TestSuccess() {
	app := NewMockApp()
	gt1 := withDefaultClearImpersonationRequest(newGinTest(s.T()))
	rt := withSetupRootTenant(app)
	u1 := newTestUser(rt)
	withSetupQueryUser2(app, u1)
	withSetupJWT(gt1.c, rt, u1)
	app.cogIdp.EXPECT().AdminUpdateUserAttributes(
		gt1.c.Request.Context(), mock.Anything, mock.Anything).Return(
		nil, nil)
	app.cogIdp.EXPECT().ListUserPoolClients(
		gt1.c.Request.Context(), mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolClientsOutput{
			UserPoolClients: []types.UserPoolClientDescription{
				{ClientId: aws.String("test-client-id1")},
			}}, nil).Once()
	app.cogIdp.EXPECT().DescribeUserPoolClient(
		gt1.c.Request.Context(), mock.Anything).Return(
		&cognitoidentityprovider.DescribeUserPoolClientOutput{
			UserPoolClient: &types.UserPoolClientType{
				ClientId:     aws.String("test-client-id1"),
				ClientSecret: aws.String("test-client-secret")}}, nil).Once()
	app.cogIdp.EXPECT().InitiateAuth(
		gt1.c.Request.Context(), mock.Anything).Return(
		&cognitoidentityprovider.InitiateAuthOutput{
			AuthenticationResult: &types.AuthenticationResultType{
				IdToken:     aws.String("id-token"),
				AccessToken: aws.String("access-token")}}, nil).Once()
	err := handleClearImpersonation(app, gt1.c)
	s.NoError(err)
	resp := ClearImpersonationResponse{}
	MustJSONUnmarshal(gt1.w.Body.Bytes(), &resp)
	s.Equal("id-token", resp.IdToken)
	s.Equal("access-token", resp.AccessToken)
}
