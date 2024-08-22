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
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/stretchr/testify/suite"
)

type HandleClearImpersonateTestSuite struct {
	suite.Suite
	ginMode string
}

func TestHandleClearImpersonateTestSuite(t *testing.T) {
	suite.Run(t, &HandleClearImpersonateTestSuite{})
}

func (s *HandleClearImpersonateTestSuite) SetupTest() {
	s.ginMode = gin.Mode()
	gin.SetMode(gin.TestMode)
}

func (s *HandleClearImpersonateTestSuite) TearDownTest() {
	gin.SetMode(s.ginMode)
}

func withDefaultClearImpersonateRequest(gt *ginTest) *ginTest {
	return gt.withRequest(
		httptest.NewRequest(
			"POST",
			"/clear-impersonate",
			bytes.NewBuffer(MustJSONMarshal(
				ClearImpersonateRequest{RefreshToken: "refresh_token"}))))
}

func (s *HandleClearImpersonateTestSuite) TestBadRequests() {
	app := NewMockApp()
	gt1 := newGinTest(s.T()).withRequest(
		httptest.NewRequest("POST", "/clear-impersonate", nil))
	err := handleClearImpersonate(app, gt1.c)
	s.Error(err)

	gt2 := newGinTest(s.T()).withRequest(
		httptest.NewRequest(
			"POST", "/clear-impersonate", bytes.NewBuffer([]byte("{}"))))
	err = handleClearImpersonate(app, gt2.c)
	s.Error(err)

	gt3 := withDefaultClearImpersonateRequest(newGinTest(s.T()))
	err = handleClearImpersonate(app, gt3.c)
	s.Error(err)

	gt4 := withDefaultClearImpersonateRequest(newGinTest(s.T()))
	gt4.c.Set(TokenKey, "not-the-right-type")
	err = handleClearImpersonate(app, gt4.c)
	s.Error(err)
}

func (s *HandleClearImpersonateTestSuite) TestBadTokens() {
	app := NewMockApp()
	gt1 := withDefaultClearImpersonateRequest(newGinTest(s.T()))
	gt1.c.Set(TokenKey, &jwt.Token{})
	err := handleClearImpersonate(app, gt1.c)
	s.Error(err)

	gt2 := withDefaultClearImpersonateRequest(newGinTest(s.T()))
	claims := make(jwt.MapClaims)
	claims[authx.CognitoGroupsClaim] = []string{"group1", "group2"}
	tok := &jwt.Token{Claims: claims}
	gt2.c.Set(TokenKey, tok)
	err = handleClearImpersonate(app, gt2.c)
	s.Error(err)
}

func (s *HandleClearImpersonateTestSuite) TestNoRootTenant() {
	app := NewMockApp()
	gt1 := withDefaultClearImpersonateRequest(newGinTest(s.T()))
	claims := make(jwt.MapClaims)
	tenantUUID := uuid.New().String()
	claims[authx.CognitoGroupsClaim] = []string{
		fmt.Sprintf("tenant-%s-guest", tenantUUID)}
	tok := &jwt.Token{Claims: claims}
	gt1.c.Set(TokenKey, tok)
	app.ddb.EXPECT().Query(mock.Anything, mock.Anything).Return(
		nil, fmt.Errorf("error"))
	err := handleClearImpersonate(app, gt1.c)
	s.Error(err)
}

func (s *HandleClearImpersonateTestSuite) TestNotAuthorizedForRoot() {
	app := NewMockApp()
	gt1 := withDefaultClearImpersonateRequest(newGinTest(s.T()))
	claims := make(jwt.MapClaims)
	tenantUUID := uuid.New().String()
	claims[authx.CognitoGroupsClaim] = []string{
		fmt.Sprintf("tenant-%s-guest", tenantUUID)}
	tok := &jwt.Token{Claims: claims}
	gt1.c.Set(TokenKey, tok)
	o := dynamodb.QueryOutput{}
	mt := multi.NewTenant()
	o.Items = append(o.Items, MustStructToDynamoMap(mt))
	app.ddb.EXPECT().Query(mock.Anything, mock.Anything).Return(&o, nil)
	err := handleClearImpersonate(app, gt1.c)
	s.Error(err)
}

func (s *HandleClearImpersonateTestSuite) TestNoEmailClaim() {
	app := NewMockApp()
	gt1 := withDefaultClearImpersonateRequest(newGinTest(s.T()))
	mt := multi.NewTenant()
	claims := make(jwt.MapClaims)
	claims[authx.CognitoGroupsClaim] = []string{
		fmt.Sprintf("tenant-%s-guest", mt.ID.String())}
	tok := &jwt.Token{Claims: claims}
	gt1.c.Set(TokenKey, tok)
	o := dynamodb.QueryOutput{}
	o.Items = append(o.Items, MustStructToDynamoMap(mt))
	app.ddb.EXPECT().Query(mock.Anything, mock.Anything).Return(&o, nil)
	err := handleClearImpersonate(app, gt1.c)
	s.Error(err)
}

func (s *HandleClearImpersonateTestSuite) TestFailedToUpdateUserAttributes() {
	app := NewMockApp()
	gt1 := withDefaultClearImpersonateRequest(newGinTest(s.T()))
	mt := multi.NewTenant()
	claims := make(jwt.MapClaims)
	claims[authx.CognitoGroupsClaim] = []string{
		fmt.Sprintf("tenant-%s-guest", mt.ID.String())}
	claims[authx.EmailClaim] = "foo@example.com"
	tok := &jwt.Token{Claims: claims}
	gt1.c.Set(TokenKey, tok)
	o := dynamodb.QueryOutput{}
	o.Items = append(o.Items, MustStructToDynamoMap(mt))
	app.ddb.EXPECT().Query(mock.Anything, mock.Anything).Return(&o, nil)
	app.cogIdp.EXPECT().AdminUpdateUserAttributes(mock.Anything, mock.Anything).Return(
		nil, errors.New("could not update user attributes"))
	err := handleClearImpersonate(app, gt1.c)
	s.Error(err)
}

func (s *HandleClearImpersonateTestSuite) TestNoSubClaim() {
	app := NewMockApp()
	gt1 := withDefaultClearImpersonateRequest(newGinTest(s.T()))
	mt := multi.NewTenant()
	claims := make(jwt.MapClaims)
	claims[authx.CognitoGroupsClaim] = []string{
		fmt.Sprintf("tenant-%s-guest", mt.ID.String())}
	claims[authx.EmailClaim] = "foo@example.com"
	tok := &jwt.Token{Claims: claims}
	gt1.c.Set(TokenKey, tok)
	o := dynamodb.QueryOutput{}
	o.Items = append(o.Items, MustStructToDynamoMap(mt))
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
	err := handleClearImpersonate(app, gt1.c)
	s.Error(err)
}

func (s *HandleClearImpersonateTestSuite) TestSuccess() {
	app := NewMockApp()
	gt1 := withDefaultClearImpersonateRequest(newGinTest(s.T()))
	mt := multi.NewTenant()
	claims := make(jwt.MapClaims)
	claims[authx.CognitoGroupsClaim] = []string{
		fmt.Sprintf("tenant-%s-guest", mt.ID.String())}
	claims[authx.EmailClaim] = "foo@example.com"
	claims[authx.SubClaim] = "sub-123"
	tok := &jwt.Token{Claims: claims}
	gt1.c.Set(TokenKey, tok)
	o := dynamodb.QueryOutput{}
	o.Items = append(o.Items, MustStructToDynamoMap(mt))
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
	err := handleClearImpersonate(app, gt1.c)
	s.NoError(err)
	resp := ClearImpersonateResponse{}
	MustJSONUnmarshal(gt1.w.Body.Bytes(), &resp)
	s.Equal("id-token", resp.IdToken)
	s.Equal("access-token", resp.AccessToken)
}
