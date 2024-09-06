package authx_test

import (
	"context"
	"testing"

	"github.com/apresdev/landlord/internal/apptest"
	"github.com/apresdev/landlord/internal/multi"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/stretchr/testify/mock"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/apresdev/landlord/internal/mocks"
	"github.com/stretchr/testify/suite"
)

type TenantsTestSuite struct {
	suite.Suite
}

func TestTenantsTestSuite(t *testing.T) {
	suite.Run(t, &TenantsTestSuite{})
}

func (s *TenantsTestSuite) TestSetTenantStateOnRoot() {
	ctx := context.Background()
	ddb := mocks.NewAWSDynamoDB(s.T())
	t := multi.NewTenant()
	t.Name = "root"
	o := dynamodb.QueryOutput{}
	o.Items = append(o.Items, apptest.MustStructToDynamoMap(t))
	ddb.EXPECT().Query(ctx, mock.Anything).Return(&o, nil)
	th, err := multi.ParseTenantHandle("root")
	s.Assert().NoError(err)

	v1, err := authx.SetTenantState(ctx, ddb, th, multi.TenantStateActive)
	s.Assert().Error(err)
	s.Assert().Nil(v1)

	v2, err := authx.SetTenantState(ctx, ddb, th, multi.TenantStateInactive)
	s.Assert().Error(err)
	s.Assert().Empty(v2)
}

func (s *TenantsTestSuite) TestSetTenantStateOnRegularTenant() {
	ctx := context.Background()
	ddb := mocks.NewAWSDynamoDB(s.T())
	t := multi.NewTenant()
	t.Name = "regular"
	o := dynamodb.QueryOutput{}
	o.Items = append(o.Items, apptest.MustStructToDynamoMap(t))
	ddb.EXPECT().Query(ctx, mock.Anything).Return(&o, nil)
	ddb.EXPECT().PutItem(ctx, mock.Anything).Return(nil, nil)
	th, err := multi.ParseTenantHandle("regular")
	s.Assert().NoError(err)

	v1, err := authx.SetTenantState(ctx, ddb, th, multi.TenantStateActive)
	s.Assert().NoError(err)
	s.Assert().NotNil(v1)
	s.Assert().Equal(multi.TenantStateActive, v1.State)

	v2, err := authx.SetTenantState(ctx, ddb, th, multi.TenantStateInactive)
	s.Assert().NoError(err)
	s.Assert().NotNil(v2)
	s.Assert().Equal(multi.TenantStateInactive, v2.State)
}
