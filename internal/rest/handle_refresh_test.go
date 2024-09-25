package rest

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/apresdev/landlord/internal/db"

	"github.com/apresdev/landlord/internal/apperr"
	"github.com/apresdev/landlord/internal/apptest"
	"github.com/apresdev/landlord/internal/multi"
	"github.com/apresdev/landlord/pkg/client"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/mock"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
)

type HandleRefreshTestSuite struct {
	suite.Suite
	ginMode string
}

func TestHandleRefreshTestSuite(t *testing.T) {
	suite.Run(t, &HandleRefreshTestSuite{})
}

func (s *HandleRefreshTestSuite) SetupTest() {
	s.ginMode = gin.Mode()
	gin.SetMode(gin.TestMode)
}

func (s *HandleRefreshTestSuite) TearDownTest() {
	gin.SetMode(s.ginMode)
}

type badRequest struct {
	Username int `json:"username"`
}

func fakeUserRequest() client.RefreshRequest {
	return client.RefreshRequest{
		Username:     "user@example.com",
		RefreshToken: "fake-refresh-token",
	}
}

func newTestUser(username string, tenantID uuid.UUID) *multi.User {
	return &multi.User{
		Username: username,
		TenantID: tenantID,
	}
}

func (s *HandleRefreshTestSuite) TestBadRequests() {
	app := NewMockApp()
	gt1 := newGinTest(s.T()).withRequest(
		httptest.NewRequest("POST", "/refresh", nil))
	err := handleRefresh(app, gt1.c)
	s.True(apperr.IsBadRequest(err))

	b, err := json.Marshal(badRequest{Username: 1})
	s.NoError(err)
	gt2 := newGinTest(s.T()).withRequest(
		httptest.NewRequest("POST", "/refresh", bytes.NewReader(b)))
	err = handleRefresh(app, gt2.c)
	s.True(apperr.IsBadRequest(err))
}

func (s *HandleRefreshTestSuite) TestNoRefreshToken() {
	req := client.RefreshRequest{Username: "test@example.com"}
	b, err := json.Marshal(req)
	s.NoError(err)
	app := NewMockApp()
	gt1 := newGinTest(s.T()).withRequest(
		httptest.NewRequest("POST", "/refresh", bytes.NewReader(b)))
	err = handleRefresh(app, gt1.c)
	s.True(apperr.IsBadRequest(err))
}

func (s *HandleRefreshTestSuite) TestUserNotFound() {
	req := fakeUserRequest()
	b, err := json.Marshal(req)
	s.NoError(err)
	app := NewMockApp()
	app.ddb.EXPECT().Query(mock.Anything, mock.Anything).Return(
		nil, apperr.NewNotFound("not found"))
	gt1 := newGinTest(s.T()).withRequest(
		httptest.NewRequest("POST", "/refresh", bytes.NewReader(b)))
	err = handleRefresh(app, gt1.c)
	s.True(apperr.IsUnauthorized(err))
}

func (s *HandleRefreshTestSuite) TestNoUserPoolClient() {
	req := fakeUserRequest()
	b, err := json.Marshal(req)
	s.NoError(err)
	app := NewMockApp()
	mt := multi.NewTenant()
	mu := newTestUser("user@example.com", mt.ID)

	// User is queried first...
	o1 := dynamodb.QueryOutput{}
	o1.Items = append(o1.Items, apptest.MustStructToDynamoMap(*mu))
	app.ddb.EXPECT().Query(mock.Anything, mock.Anything).Return(&o1, nil).Once()

	// ...then tenant
	o2 := dynamodb.GetItemOutput{}
	o2.Item = apptest.MustStructToDynamoMap(*mt)
	app.ddb.EXPECT().GetItem(mock.Anything, mock.Anything).Return(&o2, nil).Once()

	app.cogIdp.EXPECT().ListUserPoolClients(mock.Anything, mock.Anything).Return(
		nil, apperr.NewNotFound("not found"))

	gt1 := newGinTest(s.T()).withRequest(
		httptest.NewRequest("POST", "/refresh", bytes.NewReader(b)))
	err = handleRefresh(app, gt1.c)
	s.True(apperr.IsUnauthorized(err))
}

func (s *HandleRefreshTestSuite) TestUserLookupFailure() {
	req := fakeUserRequest()
	b, err := json.Marshal(req)
	s.NoError(err)
	app := NewMockApp()
	mt := multi.NewTenant()
	mu := newTestUser("user@example.com", mt.ID)

	// User is queried first...
	o1 := dynamodb.QueryOutput{}
	o1.Items = append(o1.Items, apptest.MustStructToDynamoMap(*mu))
	app.ddb.EXPECT().Query(mock.Anything, mock.Anything).Return(&o1, nil).Once()

	// ...then tenant
	o2 := dynamodb.GetItemOutput{}
	o2.Item = apptest.MustStructToDynamoMap(*mt)
	app.ddb.EXPECT().GetItem(mock.Anything, mock.Anything).Return(&o2, nil).Once()

	app.cogIdp.EXPECT().ListUserPoolClients(mock.Anything, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolClientsOutput{
			UserPoolClients: []types.UserPoolClientDescription{
				{ClientId: aws.String("test-client-id1")},
			}}, nil).Once()
	app.cogIdp.EXPECT().DescribeUserPoolClient(mock.Anything, mock.Anything).Return(
		&cognitoidentityprovider.DescribeUserPoolClientOutput{
			UserPoolClient: &types.UserPoolClientType{
				ClientId:     aws.String("test-client-id1"),
				ClientSecret: aws.String("test-client-secret")}}, nil).Once()
	app.cogIdp.EXPECT().AdminGetUser(mock.Anything, mock.Anything).Return(
		nil, apperr.NewNotFound("not found")).Once()
	gt1 := newGinTest(s.T()).withRequest(
		httptest.NewRequest("POST", "/refresh", bytes.NewReader(b)))
	err = handleRefresh(app, gt1.c)
	s.True(apperr.IsUnauthorized(err))
}

func (s *HandleRefreshTestSuite) TestNoSubInUser() {
	req := fakeUserRequest()
	b, err := json.Marshal(req)
	s.NoError(err)
	app := NewMockApp()
	mt := multi.NewTenant()
	mu := newTestUser("user@example.com", mt.ID)

	// User is queried first...
	o1 := dynamodb.QueryOutput{}
	o1.Items = append(o1.Items, apptest.MustStructToDynamoMap(*mu))
	app.ddb.EXPECT().Query(mock.Anything, mock.Anything).Return(&o1, nil).Once()

	// ...then tenant
	o2 := dynamodb.GetItemOutput{}
	o2.Item = apptest.MustStructToDynamoMap(*mt)
	app.ddb.EXPECT().GetItem(mock.Anything, mock.Anything).Return(&o2, nil).Once()

	app.cogIdp.EXPECT().ListUserPoolClients(mock.Anything, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolClientsOutput{
			UserPoolClients: []types.UserPoolClientDescription{
				{ClientId: aws.String("test-client-id1")},
			}}, nil).Once()
	app.cogIdp.EXPECT().DescribeUserPoolClient(mock.Anything, mock.Anything).Return(
		&cognitoidentityprovider.DescribeUserPoolClientOutput{
			UserPoolClient: &types.UserPoolClientType{
				ClientId:     aws.String("test-client-id1"),
				ClientSecret: aws.String("test-client-secret")}}, nil).Once()
	u := cognitoidentityprovider.AdminGetUserOutput{
		Username: aws.String("user@example.com"),
	}
	app.cogIdp.EXPECT().AdminGetUser(mock.Anything, mock.Anything).Return(
		&u, nil).Once()

	gt1 := newGinTest(s.T()).withRequest(
		httptest.NewRequest("POST", "/refresh", bytes.NewReader(b)))
	err = handleRefresh(app, gt1.c)
	s.True(apperr.IsUnauthorized(err))
}

func (s *HandleRefreshTestSuite) TestInitiateAuthFailure() {
	req := fakeUserRequest()
	b, err := json.Marshal(req)
	s.NoError(err)
	app := NewMockApp()
	mt := multi.NewTenant()
	mu := newTestUser("user@example.com", mt.ID)

	// User is queried first...
	o1 := dynamodb.QueryOutput{}
	o1.Items = append(o1.Items, apptest.MustStructToDynamoMap(*mu))
	app.ddb.EXPECT().Query(mock.Anything, mock.Anything).Return(&o1, nil).Once()

	// ...then tenant
	o2 := dynamodb.GetItemOutput{}
	o2.Item = apptest.MustStructToDynamoMap(*mt)
	app.ddb.EXPECT().GetItem(mock.Anything, mock.Anything).Return(&o2, nil).Once()

	app.cogIdp.EXPECT().ListUserPoolClients(mock.Anything, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolClientsOutput{
			UserPoolClients: []types.UserPoolClientDescription{
				{ClientId: aws.String("test-client-id1")},
			}}, nil).Once()
	app.cogIdp.EXPECT().DescribeUserPoolClient(mock.Anything, mock.Anything).Return(
		&cognitoidentityprovider.DescribeUserPoolClientOutput{
			UserPoolClient: &types.UserPoolClientType{
				ClientId:     aws.String("test-client-id1"),
				ClientSecret: aws.String("test-client-secret")}}, nil).Once()
	u := cognitoidentityprovider.AdminGetUserOutput{
		Username: aws.String("user@example.com"),
		UserAttributes: []types.AttributeType{
			{Name: aws.String("sub"), Value: aws.String("sub-123")},
		},
	}
	app.cogIdp.EXPECT().AdminGetUser(mock.Anything, mock.Anything).Return(
		&u, nil).Once()

	app.cogIdp.EXPECT().InitiateAuth(mock.Anything, mock.Anything).Return(
		nil, apperr.NewInternal("unauthorized")).Once()

	gt1 := newGinTest(s.T()).withRequest(
		httptest.NewRequest("POST", "/refresh", bytes.NewReader(b)))
	err = handleRefresh(app, gt1.c)
	s.True(apperr.IsUnauthorized(err))
}

func (s *HandleRefreshTestSuite) TestHappyPath() {
	app := NewMockApp()

	mt := multi.NewTenant()

	// User is queried first...
	username := "user@example.com"
	usernameUUID := uuid.New()
	kv := make(map[string]ddbtypes.AttributeValue)
	kv[":altName"] = &ddbtypes.AttributeValueMemberS{Value: db.UserEntityID(username)}
	i1 := &dynamodb.QueryInput{
		TableName:                 aws.String(db.TableName()),
		IndexName:                 aws.String("altNameIndex"),
		KeyConditionExpression:    aws.String("altName = :altName"),
		ExpressionAttributeValues: kv,
	}
	o1 := dynamodb.QueryOutput{}
	item := make(map[string]ddbtypes.AttributeValue)
	item["pk"] = &ddbtypes.AttributeValueMemberS{Value: db.TenantEntityID(mt.ID.String())}
	item["sk"] = &ddbtypes.AttributeValueMemberS{Value: db.UserEntityID(usernameUUID.String())}
	item["altName"] = &ddbtypes.AttributeValueMemberS{Value: db.UserEntityID(username)}
	o1.Items = append(o1.Items, item)
	app.ddb.On("Query", mock.Anything, i1).Return(&o1, nil)

	// ...then tenant
	o2 := dynamodb.GetItemOutput{}
	o2.Item = apptest.MustStructToDynamoMap(*mt)
	o2.Item["pk"] = &ddbtypes.AttributeValueMemberS{Value: db.GlobalEntityID("*")}
	o2.Item["sk"] = &ddbtypes.AttributeValueMemberS{Value: db.TenantEntityID(mt.ID.String())}
	o2.Item["altName"] = &ddbtypes.AttributeValueMemberS{Value: db.TenantEntityID(mt.Name)}
	app.ddb.EXPECT().GetItem(mock.Anything, mock.Anything).Return(&o2, nil)

	app.cogIdp.EXPECT().ListUserPoolClients(mock.Anything, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolClientsOutput{
			UserPoolClients: []types.UserPoolClientDescription{
				{ClientId: aws.String("test-client-id1")},
			}}, nil).Once()
	app.cogIdp.EXPECT().DescribeUserPoolClient(mock.Anything, mock.Anything).Return(
		&cognitoidentityprovider.DescribeUserPoolClientOutput{
			UserPoolClient: &types.UserPoolClientType{
				ClientId:     aws.String("test-client-id1"),
				ClientSecret: aws.String("test-client-secret")}}, nil).Once()
	u := cognitoidentityprovider.AdminGetUserOutput{
		Username: aws.String("user@example.com"),
		UserAttributes: []types.AttributeType{
			{Name: aws.String("sub"), Value: aws.String("sub-123")},
		},
	}
	app.cogIdp.EXPECT().AdminGetUser(mock.Anything, mock.Anything).Return(
		&u, nil).Once()
	o3 := cognitoidentityprovider.InitiateAuthOutput{
		AuthenticationResult: &types.AuthenticationResultType{
			IdToken:     aws.String("id-token"),
			AccessToken: aws.String("access-token"),
		},
	}
	app.cogIdp.EXPECT().InitiateAuth(mock.Anything, mock.Anything).Return(
		&o3, nil).Once()

	req := client.RefreshRequest{}
	req.Username = "user@example.com"
	req.RefreshToken = "fake-refresh-token"
	b, err := json.Marshal(req)
	s.NoError(err)
	gt1 := newGinTest(s.T()).withRequest(
		httptest.NewRequest("POST", "/refresh", bytes.NewReader(b)))

	err = handleRefresh(app, gt1.c)
	s.NoError(err)

	bodyBytes, err := io.ReadAll(gt1.w.Result().Body)
	s.NoError(err)
	resp := client.LoginResponse{}
	err = json.Unmarshal(bodyBytes, &resp)
	s.NoError(err)
	s.Equal("id-token", resp.IdToken)
	s.Equal("access-token", resp.AccessToken)
}
