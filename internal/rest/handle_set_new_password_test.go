package rest

import (
	"bytes"
	"context"
	"net/http/httptest"
	"testing"

	"github.com/apresdev/landlord/internal/db"
	"github.com/apresdev/landlord/internal/multi"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/mock"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/suite"
)

type HandleSetNewPasswordTestSuite struct {
	suite.Suite
	ginMode string
}

func TestHandleSetNewPasswordTestSuite(t *testing.T) {
	suite.Run(t, &HandleSetNewPasswordTestSuite{})
}

func (s *HandleSetNewPasswordTestSuite) SetupTest() {
	s.ginMode = gin.Mode()
	gin.SetMode(gin.TestMode)
}

func (s *HandleSetNewPasswordTestSuite) TearDownTest() {
	gin.SetMode(s.ginMode)
}

func withDefaultSetNewPasswordRequest(gt *ginTest) *ginTest {
	return withDefaultSetNewPasswordRequest2(gt, &SetNewPasswordRequest{
		Username: "user1@example.com",
		Password: "password",
		Session:  "session-token"})
}

func withDefaultSetNewPasswordRequest2(gt *ginTest, req *SetNewPasswordRequest) *ginTest {
	gt.c.Set(TokenKey, &jwt.Token{})
	return gt.withRequest(
		httptest.NewRequest(
			"POST",
			"/new-password",
			bytes.NewBuffer(MustJSONMarshal(req))))
}

func (s *HandleSetNewPasswordTestSuite) TestBadRequests() {
	app := NewMockApp()
	gt1 := newGinTest(s.T()).withRequest(
		httptest.NewRequest("POST", "/new-password", nil))
	err := handleSetNewPassword(app, gt1.c)
	s.Error(err)

	gt2 := newGinTest(s.T()).withRequest(
		httptest.NewRequest(
			"POST", "/new-password", bytes.NewBuffer([]byte("{}"))))
	err = handleSetNewPassword(app, gt2.c)
	s.Error(err)

	gt3 := newGinTest(s.T()).withRequest(
		httptest.NewRequest(
			"POST", "/new-password", bytes.NewBuffer(MustJSONMarshal(
				SetNewPasswordRequest{Username: "user1@example.com"}))))
	err = handleSetNewPassword(app, gt3.c)
	s.Error(err)

	gt4 := newGinTest(s.T()).withRequest(
		httptest.NewRequest(
			"POST", "/new-password", bytes.NewBuffer(MustJSONMarshal(
				SetNewPasswordRequest{
					Username: "user1@example.com",
					Password: "password2"}))))
	err = handleSetNewPassword(app, gt4.c)
	s.Error(err)
}

func (s *HandleSetNewPasswordTestSuite) TestSuccess() {
	app := NewMockApp()
	ctx := context.Background()
	req := SetNewPasswordRequest{
		Username: "user1@example.com",
		Password: "password",
		Session:  "session-token"}
	gt1 := withDefaultSetNewPasswordRequest2(newGinTest(s.T()), &req)
	t := withSetupGetTenantByID(app)
	u := newUser(t)
	u.Status = multi.UserStatusInvited
	withSetupQueryUser2(app, u)
	withGetUserPoolClient(app, ctx)

	app.cogIdp.EXPECT().RespondToAuthChallenge(ctx, mock.Anything).Return(
		&cognitoidentityprovider.RespondToAuthChallengeOutput{
			AuthenticationResult: &types.AuthenticationResultType{
				AccessToken:  aws.String("access-token"),
				IdToken:      aws.String("id-token"),
				RefreshToken: aws.String("refresh-token"),
			},
		}, nil).Once()
	app.cogIdp.EXPECT().AdminConfirmSignUp(gt1.c.Request.Context(), mock.Anything).Return(
		&cognitoidentityprovider.AdminConfirmSignUpOutput{}, nil).Once()
	app.cogIdp.EXPECT().AdminUpdateUserAttributes(gt1.c.Request.Context(), mock.Anything).Return(
		&cognitoidentityprovider.AdminUpdateUserAttributesOutput{}, nil).Once()
	app.cogIdp.EXPECT().AdminAddUserToGroup(gt1.c.Request.Context(), mock.Anything).Return(
		&cognitoidentityprovider.AdminAddUserToGroupOutput{}, nil).Once()

	av, err := attributevalue.MarshalMap(u)
	s.NoError(err)
	av["pk"] = &ddbtypes.AttributeValueMemberS{Value: db.TenantEntityID(u.TenantID.String())}
	av["sk"] = &ddbtypes.AttributeValueMemberS{Value: db.UserEntityID(u.ID.String())}
	av["altName"] = &ddbtypes.AttributeValueMemberS{Value: db.UserEntityID(u.Username)}

	// The modified value
	av["status"] = &ddbtypes.AttributeValueMemberS{Value: "active"}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(db.TableName()),
		Item:      av,
	}
	app.ddb.EXPECT().PutItem(gt1.c.Request.Context(), input).Return(
		&dynamodb.PutItemOutput{}, nil).Once()

	resp, err := HandleSetNewPassword(app, gt1.c, req)
	s.NoError(err)
	s.NotNil(resp)
	s.Equal("access-token", resp.AccessToken)
	s.Equal("id-token", resp.IdToken)
	s.Equal("refresh-token", resp.RefreshToken)
}

func (s *HandleSetNewPasswordTestSuite) TestSuccessWithMFA() {
	// When the policy is to attach an MFA token, the user should not
	// transition to active until that has been done.

	app := NewMockApp()
	req := SetNewPasswordRequest{
		Username: "user1@example.com",
		Password: "password",
		Session:  "session-token"}
	gt1 := withDefaultSetNewPasswordRequest2(newGinTest(s.T()), &req)
	t := withSetupGetTenantByID(app)
	u := newUser(t)
	u.Status = multi.UserStatusInvited
	u.MFAPolicy = multi.MFAPolicyRequired
	withSetupQueryUser2(app, u)
	withGetUserPoolClient(app, gt1.c.Request.Context())

	app.cogIdp.EXPECT().RespondToAuthChallenge(gt1.c.Request.Context(), mock.Anything).Return(
		&cognitoidentityprovider.RespondToAuthChallengeOutput{
			AuthenticationResult: &types.AuthenticationResultType{
				AccessToken: aws.String("access-token"),
				IdToken:     aws.String("id-token"),
			},
		}, nil).Once()
	app.cogIdp.EXPECT().AdminConfirmSignUp(gt1.c.Request.Context(), mock.Anything).Return(
		&cognitoidentityprovider.AdminConfirmSignUpOutput{}, nil).Once()
	app.cogIdp.EXPECT().AdminUpdateUserAttributes(gt1.c.Request.Context(), mock.Anything).Return(
		&cognitoidentityprovider.AdminUpdateUserAttributesOutput{}, nil).Once()
	app.cogIdp.EXPECT().AdminAddUserToGroup(gt1.c.Request.Context(), mock.Anything).Return(
		&cognitoidentityprovider.AdminAddUserToGroupOutput{}, nil).Once()

	app.cogIdp.EXPECT().AdminGetUser(gt1.c.Request.Context(), mock.Anything).Return(
		&cognitoidentityprovider.AdminGetUserOutput{}, nil).Once()
	app.cogIdp.EXPECT().AssociateSoftwareToken(gt1.c.Request.Context(), mock.Anything).Return(
		&cognitoidentityprovider.AssociateSoftwareTokenOutput{
			SecretCode: aws.String("secret-code"),
		}, nil).Once()

	resp, err := HandleSetNewPassword(app, gt1.c, req)
	s.NoError(err)
	s.NotNil(resp)
	s.Equal(LoginStatusMFASetupRequired, resp.Status)
	s.NotEmpty(resp.OtpURI)
	s.Equal("access-token", resp.AccessToken)
	s.Empty(resp.IdToken)
	s.Empty(resp.RefreshToken)
}
