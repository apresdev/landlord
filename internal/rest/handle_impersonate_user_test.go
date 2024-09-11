package rest

import (
	"bytes"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/apresdev/landlord/internal/apptest"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/apresdev/landlord/internal/multi"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/stretchr/testify/suite"
)

type HandleImpersonateUserTestSuite struct {
	suite.Suite
	ginMode string
}

func TestHandleImpersonateUserTestSuite(t *testing.T) {
	suite.Run(t, &HandleImpersonateUserTestSuite{})
}

func (s *HandleImpersonateUserTestSuite) SetupTest() {
	s.ginMode = gin.Mode()
	gin.SetMode(gin.TestMode)
}

func (s *HandleImpersonateUserTestSuite) TearDownTest() {
	gin.SetMode(s.ginMode)
}

func withDefaultImpersonateUserRequest(gt *ginTest) *ginTest {
	return gt.withRequest(
		httptest.NewRequest(
			"POST",
			"/impersonate-user",
			bytes.NewBuffer(MustJSONMarshal(
				ImpersonateUserRequest{
					Tenant:       "tenant-to-impersonate",
					Username:     "impersonate@example.com",
					RefreshToken: "refresh_token"}))))
}

func (s *HandleImpersonateUserTestSuite) TestBadRequests() {
	app := NewMockApp()
	gt1 := newGinTest(s.T()).withRequest(
		httptest.NewRequest("POST", "/impersonate-user", nil))
	err := handleImpersonateUser(app, gt1.c)
	s.Error(err)

	gt2 := newGinTest(s.T()).withRequest(
		httptest.NewRequest(
			"POST", "/impersonate-user", bytes.NewBuffer([]byte("{}"))))
	err = handleImpersonateUser(app, gt2.c)
	s.Error(err)

	gt3 := newGinTest(s.T()).withRequest(
		httptest.NewRequest(
			"POST", "/impersonate-user", bytes.NewBuffer(MustJSONMarshal(
				ImpersonateUserRequest{Tenant: "tenant-to-impersonate"}))))
	err = handleImpersonateUser(app, gt3.c)
	s.Error(err)

	gt4 := newGinTest(s.T()).withRequest(
		httptest.NewRequest(
			"POST", "/impersonate-user", bytes.NewBuffer(MustJSONMarshal(
				ImpersonateUserRequest{
					Tenant:   "tenant-to-impersonate",
					Username: "impersonate@example.com"}))))
	err = handleImpersonateUser(app, gt4.c)
	s.Error(err)

	gt5 := withDefaultImpersonateUserRequest(newGinTest(s.T()))
	err = handleImpersonateUser(app, gt5.c)
	s.Error(err)

	gt6 := withDefaultImpersonateUserRequest(newGinTest(s.T()))
	gt6.c.Set(TokenKey, "not-the-right-type")
	err = handleImpersonateUser(app, gt6.c)
	s.Error(err)
}

func (s *HandleImpersonateUserTestSuite) TestBadTokens() {
	app := NewMockApp()
	gt1 := withDefaultImpersonateUserRequest(newGinTest(s.T()))
	gt1.c.Set(TokenKey, &jwt.Token{})
	err := handleImpersonateUser(app, gt1.c)
	s.Error(err)

	gt2 := withDefaultImpersonateUserRequest(newGinTest(s.T()))
	claims := make(jwt.MapClaims)
	claims[authx.CognitoGroupsClaim] = []string{"group1", "group2"}
	tok := &jwt.Token{Claims: claims}
	gt2.c.Set(TokenKey, tok)
	err = handleImpersonateUser(app, gt2.c)
	s.Error(err)
}

func (s *HandleImpersonateUserTestSuite) TestNoRootTenant() {
	app := NewMockApp()
	gt1 := withDefaultImpersonateUserRequest(newGinTest(s.T()))
	claims := make(jwt.MapClaims)
	tenantUUID := uuid.New().String()
	claims[authx.CognitoGroupsClaim] = []string{
		fmt.Sprintf("tenant-%s-guest", tenantUUID)}
	tok := &jwt.Token{Claims: claims}
	gt1.c.Set(TokenKey, tok)
	app.ddb.EXPECT().Query(mock.Anything, mock.Anything).Return(
		nil, fmt.Errorf("error"))
	err := handleImpersonateUser(app, gt1.c)
	s.Error(err)
}

func (s *HandleImpersonateUserTestSuite) TestNotAuthorizedForRoot() {
	app := NewMockApp()
	gt1 := withDefaultImpersonateUserRequest(newGinTest(s.T()))
	claims := make(jwt.MapClaims)
	tenantUUID := uuid.New().String()
	claims[authx.CognitoGroupsClaim] = []string{
		fmt.Sprintf("tenant-%s-guest", tenantUUID)}
	tok := &jwt.Token{Claims: claims}
	gt1.c.Set(TokenKey, tok)
	o := dynamodb.QueryOutput{}
	mt := multi.NewTenant()
	o.Items = append(o.Items, apptest.MustStructToDynamoMap(*mt))
	app.ddb.EXPECT().Query(mock.Anything, mock.Anything).Return(&o, nil)
	err := handleImpersonateUser(app, gt1.c)
	s.Error(err)
}

func (s *HandleImpersonateUserTestSuite) TestNoEmailClaim() {
	app := NewMockApp()
	gt1 := withDefaultImpersonateUserRequest(newGinTest(s.T()))
	mt := multi.NewTenant()
	claims := make(jwt.MapClaims)
	claims[authx.CognitoGroupsClaim] = []string{
		fmt.Sprintf("tenant-%s-guest", mt.ID.String())}
	tok := &jwt.Token{Claims: claims}
	gt1.c.Set(TokenKey, tok)
	o := dynamodb.QueryOutput{}
	o.Items = append(o.Items, apptest.MustStructToDynamoMap(*mt))
	app.ddb.EXPECT().Query(mock.Anything, mock.Anything).Return(&o, nil)
	err := handleImpersonateUser(app, gt1.c)
	s.Error(err)
}

func (s *HandleImpersonateUserTestSuite) TestFailedToUpdateUserAttributes() {
	app := NewMockApp()
	gt1 := withDefaultImpersonateUserRequest(newGinTest(s.T()))
	mt := multi.NewTenant()
	claims := make(jwt.MapClaims)
	claims[authx.CognitoGroupsClaim] = []string{
		fmt.Sprintf("tenant-%s-guest", mt.ID.String())}
	claims[authx.EmailClaim] = "foo@example.com"
	tok := &jwt.Token{Claims: claims}
	gt1.c.Set(TokenKey, tok)
	o := dynamodb.QueryOutput{}
	o.Items = append(o.Items, apptest.MustStructToDynamoMap(*mt))
	app.ddb.EXPECT().Query(mock.Anything, mock.Anything).Return(&o, nil)
	app.cogIdp.EXPECT().AdminUpdateUserAttributes(mock.Anything, mock.Anything).Return(
		nil, errors.New("could not update user attributes"))
	err := handleImpersonateUser(app, gt1.c)
	s.Error(err)
}

func (s *HandleImpersonateUserTestSuite) TestNoSubClaim() {
	app := NewMockApp()
	gt1 := withDefaultImpersonateUserRequest(newGinTest(s.T()))
	mt := multi.NewTenant()
	claims := make(jwt.MapClaims)
	claims[authx.CognitoGroupsClaim] = []string{
		fmt.Sprintf("tenant-%s-guest", mt.ID.String())}
	claims[authx.EmailClaim] = "foo@example.com"
	tok := &jwt.Token{Claims: claims}
	gt1.c.Set(TokenKey, tok)
	o := dynamodb.QueryOutput{}
	o.Items = append(o.Items, apptest.MustStructToDynamoMap(*mt))
	app.ddb.EXPECT().Query(mock.Anything, mock.Anything).Return(&o, nil)
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
	err := handleImpersonateUser(app, gt1.c)
	s.Error(err)
}

func (s *HandleImpersonateUserTestSuite) TestSuccess() {
	app := NewMockApp()
	gt1 := withDefaultImpersonateUserRequest(newGinTest(s.T()))
	mt := multi.NewTenant()
	claims := make(jwt.MapClaims)
	claims[authx.CognitoGroupsClaim] = []string{
		fmt.Sprintf("tenant-%s-guest", mt.ID.String())}
	claims[authx.EmailClaim] = "foo@example.com"
	claims[authx.SubClaim] = "sub-123"
	tok := &jwt.Token{Claims: claims}
	gt1.c.Set(TokenKey, tok)
	o := dynamodb.QueryOutput{}
	mt.Roles = append(mt.Roles, "guest")
	o.Items = append(o.Items, apptest.MustStructToDynamoMap(*mt))
	app.ddb.EXPECT().Query(mock.Anything, mock.Anything).Return(&o, nil)
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
	err := handleImpersonateUser(app, gt1.c)
	s.NoError(err)
	resp := ImpersonateResponse{}
	MustJSONUnmarshal(gt1.w.Body.Bytes(), &resp)
	s.Equal("id-token", resp.IdToken)
	s.Equal("access-token", resp.AccessToken)
}
