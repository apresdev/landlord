package authx_test

import (
	"context"
	"errors"
	"testing"

	"github.com/apresdev/landlord/internal/apperr"
	"github.com/apresdev/landlord/internal/apptest"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/stretchr/testify/mock"

	"github.com/apresdev/landlord/internal/multi"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/apresdev/landlord/internal/mocks"
	"github.com/stretchr/testify/suite"
)

type InviteTestSuite struct {
	suite.Suite
}

func TestInviteTestSuite(t *testing.T) {
	suite.Run(t, &InviteTestSuite{})
}

func (s *InviteTestSuite) TestBadEmail() {
	ctx := context.Background()
	ddb := mocks.NewAWSDynamoDB(s.T())
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	tenant := multi.NewTenant()
	err := authx.SendInvite(
		ctx, ddb, cogIdp, &tenant, "not-an-email-address", "guest", multi.MFAPolicyNone)
	s.Error(err)
	s.Equal(err.Error(), "invalid email")
}

func (s *InviteTestSuite) TestInviteDBFailure() {
	ctx := context.Background()
	ddb := mocks.NewAWSDynamoDB(s.T())
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	ddb.EXPECT().Query(mock.Anything, mock.Anything).Return(nil, errors.New("db error"))
	tenant := multi.NewTenant()
	err := authx.SendInvite(
		ctx, ddb, cogIdp, &tenant, "foo@example.com", "guest", multi.MFAPolicyNone)
	s.Error(err)
	s.Contains(err.Error(), "db error")
}

func (s *InviteTestSuite) TestInviteInactiveUser() {
	ctx := context.Background()
	ddb := mocks.NewAWSDynamoDB(s.T())
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	user := multi.User{
		Username: "foo@example.com",
		Status:   multi.UserStatusInactive,
	}
	o := dynamodb.QueryOutput{}
	o.Items = append(o.Items, apptest.MustStructToDynamoMap(user))
	ddb.EXPECT().Query(mock.Anything, mock.Anything).Return(&o, nil)
	tenant := multi.NewTenant()
	err := authx.SendInvite(
		ctx, ddb, cogIdp, &tenant, "foo@example.com", "guest", multi.MFAPolicyNone)
	s.Error(err)
	s.Equal(err.Error(), "cannot send invite, user 'foo@example.com' is inactive")
}

func (s *InviteTestSuite) TestInviteNewUser() {
	ctx := context.Background()
	ddb := mocks.NewAWSDynamoDB(s.T())
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	ddb.EXPECT().Query(mock.Anything, mock.Anything).Return(nil, apperr.NewNotFound("user not found"))
	cogIdp.EXPECT().AdminCreateUser(mock.Anything, mock.Anything).Return(nil, nil)
	ddb.EXPECT().PutItem(mock.Anything, mock.Anything).Return(nil, nil)
	tenant := multi.NewTenant()
	err := authx.SendInvite(
		ctx, ddb, cogIdp, &tenant, "foo@example.com", "guest", multi.MFAPolicyNone)
	s.NoError(err)
}
