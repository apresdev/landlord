package rest

import (
	"bytes"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/apresdev/landlord/internal/multi"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/stretchr/testify/suite"
)

type HandleImpersonateRoleTestSuite struct {
	suite.Suite
	ginMode string
}

func TestHandleImpersonateRoleTestSuite(t *testing.T) {
	suite.Run(t, &HandleImpersonateRoleTestSuite{})
}

func (s *HandleImpersonateRoleTestSuite) SetupTest() {
	s.ginMode = gin.Mode()
	gin.SetMode(gin.TestMode)
}

func (s *HandleImpersonateRoleTestSuite) TearDownTest() {
	gin.SetMode(s.ginMode)
}

func withDefaultImpersonateRoleRequest(gt *ginTest) *ginTest {
	return gt.withRequest(
		httptest.NewRequest(
			"POST",
			"/impersonate-role",
			bytes.NewBuffer(MustJSONMarshal(
				ImpersonateRoleRequest{
					Tenant:       "tenant-to-impersonate",
					Role:         "guest",
					RefreshToken: "refresh_token"}))))
}

func (s *HandleImpersonateRoleTestSuite) TestBadRequests() {
	app := NewMockApp()
	gt1 := newGinTest(s.T()).withRequest(
		httptest.NewRequest("POST", "/impersonate-role", nil))
	err := handleImpersonateRole(app, gt1.c)
	s.Error(err)

	gt2 := newGinTest(s.T()).withRequest(
		httptest.NewRequest(
			"POST", "/impersonate-role", bytes.NewBuffer([]byte("{}"))))
	err = handleImpersonateRole(app, gt2.c)
	s.Error(err)

	gt3 := newGinTest(s.T()).withRequest(
		httptest.NewRequest(
			"POST", "/impersonate-role", bytes.NewBuffer(MustJSONMarshal(
				ImpersonateRoleRequest{Tenant: "tenant-to-impersonate"}))))
	err = handleImpersonateRole(app, gt3.c)
	s.Error(err)

	gt4 := newGinTest(s.T()).withRequest(
		httptest.NewRequest(
			"POST", "/impersonate-role", bytes.NewBuffer(MustJSONMarshal(
				ImpersonateRoleRequest{
					Tenant: "tenant-to-impersonate",
					Role:   "role-to-impersonate"}))))
	err = handleImpersonateRole(app, gt4.c)
	s.Error(err)

	gt5 := withDefaultImpersonateRoleRequest(newGinTest(s.T()))
	err = handleImpersonateRole(app, gt5.c)
	s.Error(err)

	gt6 := withDefaultImpersonateRoleRequest(newGinTest(s.T()))
	gt6.c.Set(TokenKey, "not-the-right-type")
	err = handleImpersonateRole(app, gt6.c)
	s.Error(err)
}

func (s *HandleImpersonateRoleTestSuite) TestBadTokens() {
	app := NewMockApp()
	gt1 := withDefaultImpersonateRoleRequest(newGinTest(s.T()))
	gt1.c.Set(TokenKey, &jwt.Token{})
	err := handleImpersonateRole(app, gt1.c)
	s.Error(err)

	gt2 := withDefaultImpersonateRoleRequest(newGinTest(s.T()))
	claims := make(jwt.MapClaims)
	claims[authx.CognitoGroupsClaim] = []string{"group1", "group2"}
	tok := &jwt.Token{Claims: claims}
	gt2.c.Set(TokenKey, tok)
	err = handleImpersonateRole(app, gt2.c)
	s.Error(err)
}

func (s *HandleImpersonateRoleTestSuite) TestNoRootTenant() {
	app := NewMockApp()
	gt1 := withDefaultImpersonateRoleRequest(newGinTest(s.T()))
	claims := make(jwt.MapClaims)
	tenantUUID := uuid.New().String()
	claims[authx.CognitoGroupsClaim] = []string{
		fmt.Sprintf("tenant-%s-guest", tenantUUID)}
	tok := &jwt.Token{Claims: claims}
	gt1.c.Set(TokenKey, tok)
	app.ddb.EXPECT().GetItem(mock.Anything, mock.Anything).Return(
		nil, fmt.Errorf("error"))
	err := handleImpersonateRole(app, gt1.c)
	s.Error(err)
}

func (s *HandleImpersonateRoleTestSuite) TestNotAuthorizedForRoot() {
	app := NewMockApp()
	gt1 := withDefaultImpersonateRoleRequest(newGinTest(s.T()))
	claims := make(jwt.MapClaims)
	tenantUUID := uuid.New().String()
	claims[authx.CognitoGroupsClaim] = []string{
		fmt.Sprintf("tenant-%s-guest", tenantUUID)}
	tok := &jwt.Token{Claims: claims}
	gt1.c.Set(TokenKey, tok)
	_ = withSetupRootTenant(app)
	err := handleImpersonateRole(app, gt1.c)
	s.Error(err)
}

func (s *HandleImpersonateRoleTestSuite) TestNoEmailClaim() {
	app := NewMockApp()
	gt1 := withDefaultImpersonateRoleRequest(newGinTest(s.T()))
	rt := withSetupRootTenant(app)
	t := multi.NewTenant()
	t.Name = "tenant-to-impersonate"
	withSetupQueryTenant(app)
	claims := make(jwt.MapClaims)
	claims[authx.CognitoGroupsClaim] = []string{
		fmt.Sprintf("tenant-%s-guest", rt.ID.String())}
	tok := &jwt.Token{Claims: claims}
	gt1.c.Set(TokenKey, tok)
	err := handleImpersonateRole(app, gt1.c)
	s.Error(err)
}

func (s *HandleImpersonateRoleTestSuite) TestFailedToUpdateUserAttributes() {
	app := NewMockApp()
	gt1 := withDefaultImpersonateRoleRequest(newGinTest(s.T()))
	rt := withSetupRootTenant(app)
	t := multi.NewTenant()
	t.Name = "tenant-to-impersonate"
	withSetupQueryTenant(app)
	claims := make(jwt.MapClaims)
	claims[authx.CognitoGroupsClaim] = []string{
		fmt.Sprintf("tenant-%s-guest", rt.ID.String())}
	claims[authx.EmailClaim] = "foo@example.com"
	tok := &jwt.Token{Claims: claims}
	gt1.c.Set(TokenKey, tok)
	app.cogIdp.EXPECT().AdminUpdateUserAttributes(mock.Anything, mock.Anything).Return(
		nil, errors.New("could not update user attributes"))
	err := handleImpersonateRole(app, gt1.c)
	s.Error(err)
}

func (s *HandleImpersonateRoleTestSuite) TestNoSubClaim() {
	app := NewMockApp()
	gt1 := withDefaultImpersonateRoleRequest(newGinTest(s.T()))
	rt := withSetupRootTenant(app)
	t := multi.NewTenant()
	t.Name = "tenant-to-impersonate"
	withSetupQueryTenant(app)
	claims := make(jwt.MapClaims)
	claims[authx.CognitoGroupsClaim] = []string{
		fmt.Sprintf("tenant-%s-guest", rt.ID.String())}
	claims[authx.EmailClaim] = "foo@example.com"
	tok := &jwt.Token{Claims: claims}
	gt1.c.Set(TokenKey, tok)
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
	err := handleImpersonateRole(app, gt1.c)
	s.Error(err)
}

func (s *HandleImpersonateRoleTestSuite) TestSuccess() {
	app := NewMockApp()

	rt := withSetupRootTenant(app)

	t := multi.NewTenant()
	t.Name = "tenant-to-impersonate"
	t.Roles = append(t.Roles, "guest")
	withSetupQueryTenant(app)

	gt1 := withDefaultImpersonateRoleRequest(newGinTest(s.T()))
	claims := make(jwt.MapClaims)
	claims[authx.CognitoGroupsClaim] = []string{
		fmt.Sprintf("tenant-%s-guest", rt.ID.String())}
	claims[authx.EmailClaim] = "foo@example.com"
	claims[authx.SubClaim] = "sub-123"
	tok := &jwt.Token{Claims: claims}
	gt1.c.Set(TokenKey, tok)

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
	err := handleImpersonateRole(app, gt1.c)
	s.NoError(err)
	resp := ImpersonateResponse{}
	MustJSONUnmarshal(gt1.w.Body.Bytes(), &resp)
	s.Equal("id-token", resp.IdToken)
	s.Equal("access-token", resp.AccessToken)
}
