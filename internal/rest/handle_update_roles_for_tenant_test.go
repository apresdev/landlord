package rest

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/apresdev/landlord/internal/perms"

	"github.com/apresdev/landlord/internal/db"
	"github.com/apresdev/landlord/internal/multi"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/mock"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/golang-jwt/jwt/v5"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
)

type HandleUpdateRolesForTenantTestSuite struct {
	suite.Suite
	ginMode string
}

func TestHandleUpdateRolesForTenantTestSuite(t *testing.T) {
	suite.Run(t, &HandleUpdateRolesForTenantTestSuite{})
}

func (s *HandleUpdateRolesForTenantTestSuite) SetupTest() {
	s.ginMode = gin.Mode()
	gin.SetMode(gin.TestMode)
}

func (s *HandleUpdateRolesForTenantTestSuite) TearDownTest() {
	gin.SetMode(s.ginMode)
}

type badRequestStruct struct {
	Roles int
}

func testSetRolesForTenantRequest(t *testing.T, ten *multi.Tenant, user *multi.User, body []byte) *ginTest {
	r := httptest.NewRequest("PUT", "/roles", bytes.NewBuffer(body))
	gt := newGinTest(t).withRequest(r)
	if ten != nil {
		gt.c.AddParam("tenantID", ten.ID.String())
		claims := make(jwt.MapClaims)
		claims[authx.CustomTenantClaim] = ten.ID.String()
		if user != nil {
			claims[authx.CognitoGroupsClaim] = []string{ten.GroupName(user.Role)}
			claims[authx.EmailClaim] = user.Username
			claims[authx.SubClaim] = user.CognitoID.String()
			claims[authx.CustomUserClaim] = user.ID.String()
		}
		tok := &jwt.Token{Claims: claims}
		gt.c.Set(perms.TokenKey, tok)
	}
	return gt
}

func (s *HandleUpdateRolesForTenantTestSuite) TestBadRequests() {
	app := NewMockApp()
	gt1 := testSetRolesForTenantRequest(s.T(), nil, nil, nil)
	err := handleUpdateRolesForTenant(app, gt1.c)
	s.Error(err)

	gt2 := testSetRolesForTenantRequest(s.T(), nil, nil, MustJSONMarshal([]byte("{}")))
	err = handleUpdateRolesForTenant(app, gt2.c)
	s.Error(err)

	gt3 := testSetRolesForTenantRequest(s.T(), nil, nil, MustJSONMarshal(
		badRequestStruct{Roles: 1}))
	err = handleUpdateRolesForTenant(app, gt3.c)
	s.Error(err)

	gt4 := testSetRolesForTenantRequest(s.T(), nil, nil, MustJSONMarshal(
		TenantRolesRequest{Roles: []string{"role1"}}))
	gt4.c.Params = nil
	err = handleUpdateRolesForTenant(app, gt4.c)
	s.Error(err)
}

func (s *HandleUpdateRolesForTenantTestSuite) TestBadRoleNames() {
	app := NewMockApp()
	t1 := withSetupGetTenantByID(app)
	u1 := newTestUser(t1)
	gt := testSetRolesForTenantRequest(s.T(), t1, u1, MustJSONMarshal(
		TenantRolesRequest{Roles: []string{"admin", "guest", "not-a-valid-role-name-###"}}))
	withSetupQueryUsers(app, t1, []*multi.User{u1})
	app.cogIdp.EXPECT().CreateGroup(mock.Anything, mock.Anything).Return(nil, nil)
	err := handleUpdateRolesForTenant(app, gt.c)
	s.Error(err)
	s.Equal("role name must be all lowercase and contain only letters, numbers, underscores, and hyphens", err.Error())
}

func (s *HandleUpdateRolesForTenantTestSuite) TestSuccessAdd() {
	a := NewMockApp()
	t1 := withSetupGetTenantByID(a)
	u1 := newTestUser(t1)
	u1.Role = "admin"
	withSetupQueryUser2(a, u1)
	gt := testSetRolesForTenantRequest(s.T(), t1, u1, MustJSONMarshal(
		TenantRolesRequest{Roles: []string{"admin", "role1"}}))
	withSetupQueryUsers(a, t1, []*multi.User{u1})

	t1.Roles = []string{"admin", "guest", "role1"}
	av, _ := attributevalue.MarshalMap(t1)
	av["pk"] = &ddbtypes.AttributeValueMemberS{Value: db.GlobalEntityID("*")}
	av["sk"] = &ddbtypes.AttributeValueMemberS{Value: db.TenantEntityID(t1.ID.String())}
	av["altName"] = &ddbtypes.AttributeValueMemberS{Value: db.TenantEntityID(t1.Name)}
	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(a.DBCtx().TableName()),
	}
	a.ddb.EXPECT().PutItem(mock.Anything, input).Return(nil, nil)
	a.cogIdp.EXPECT().CreateGroup(mock.Anything, mock.Anything).Return(nil, nil)
	err := handleUpdateRolesForTenant(a, gt.c)
	s.NoError(err)
}

func (s *HandleUpdateRolesForTenantTestSuite) TestSuccessRemove() {
	a := NewMockApp()
	t1 := multi.NewTenant("ten1")
	t1.Roles = []string{"admin", "guest", "role1"}
	withSetupGetTenantByID2(a, t1)
	u1 := newTestUser(t1)
	u1.Role = "admin"
	withSetupQueryUser2(a, u1)
	gt := testSetRolesForTenantRequest(s.T(), t1, u1, MustJSONMarshal(
		TenantRolesRequest{Roles: []string{"admin", "guest"}}))
	withSetupQueryUsers(a, t1, []*multi.User{u1})

	t1.Roles = []string{"admin", "guest"}
	t1.SubTenantRoles = []string{"admin", "guest"}
	av, _ := attributevalue.MarshalMap(t1)
	av["pk"] = &ddbtypes.AttributeValueMemberS{Value: db.GlobalEntityID("*")}
	av["sk"] = &ddbtypes.AttributeValueMemberS{Value: db.TenantEntityID(t1.ID.String())}
	av["altName"] = &ddbtypes.AttributeValueMemberS{Value: db.TenantEntityID(t1.Name)}
	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(a.DBCtx().TableName()),
	}
	a.ddb.EXPECT().PutItem(mock.Anything, input).Return(nil, nil)
	a.cogIdp.EXPECT().DeleteGroup(mock.Anything, mock.Anything).Return(nil, nil)
	err := handleUpdateRolesForTenant(a, gt.c)
	s.NoError(err)
}

func (s *HandleUpdateRolesForTenantTestSuite) TestSuccessAddSubRoles() {
	a := NewMockApp()
	t1 := withSetupGetTenantByID(a)
	u1 := newTestUser(t1)
	u1.Role = "admin"
	withSetupQueryUser2(a, u1)
	gt := testSetRolesForTenantRequest(s.T(), t1, u1, MustJSONMarshal(
		TenantRolesRequest{Roles: []string{"admin", "guest"},
			SubTenantRoles: []string{"admin", "guest", "role2", "role1"}}))
	withSetupQueryUsers(a, t1, []*multi.User{u1})

	t1.Roles = []string{"admin", "guest"}
	t1.SubTenantRoles = []string{"admin", "guest", "role1", "role2"}
	av, _ := attributevalue.MarshalMap(t1)
	av["pk"] = &ddbtypes.AttributeValueMemberS{Value: db.GlobalEntityID("*")}
	av["sk"] = &ddbtypes.AttributeValueMemberS{Value: db.TenantEntityID(t1.ID.String())}
	av["altName"] = &ddbtypes.AttributeValueMemberS{Value: db.TenantEntityID(t1.Name)}
	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(a.DBCtx().TableName()),
	}
	a.ddb.EXPECT().PutItem(mock.Anything, input).Return(nil, nil)
	err := handleUpdateRolesForTenant(a, gt.c)
	s.NoError(err)
}

func (s *HandleUpdateRolesForTenantTestSuite) TestSuccessRemoveSubRole() {
	a := NewMockApp()
	t1 := withSetupGetTenantByID(a)
	t1.SubTenantRoles = []string{"admin", "guest", "role1", "role2"}
	u1 := newTestUser(t1)
	u1.Role = "admin"
	withSetupQueryUser2(a, u1)
	gt := testSetRolesForTenantRequest(s.T(), t1, u1, MustJSONMarshal(
		TenantRolesRequest{Roles: []string{"admin", "guest"},
			SubTenantRoles: []string{"admin", "guest", "role1"}}))
	withSetupQueryUsers(a, t1, []*multi.User{u1})

	t1.Roles = []string{"admin", "guest"}
	t1.SubTenantRoles = []string{"admin", "guest", "role1"}
	av, _ := attributevalue.MarshalMap(t1)
	av["pk"] = &ddbtypes.AttributeValueMemberS{Value: db.GlobalEntityID("*")}
	av["sk"] = &ddbtypes.AttributeValueMemberS{Value: db.TenantEntityID(t1.ID.String())}
	av["altName"] = &ddbtypes.AttributeValueMemberS{Value: db.TenantEntityID(t1.Name)}
	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(a.DBCtx().TableName()),
	}
	a.ddb.EXPECT().PutItem(mock.Anything, input).Return(nil, nil)
	err := handleUpdateRolesForTenant(a, gt.c)
	s.NoError(err)
}

func (s *HandleUpdateRolesForTenantTestSuite) TestSuccessAddDuplicate() {
	a := NewMockApp()
	t1 := withSetupGetTenantByID(a)
	u1 := newTestUser(t1)
	u1.Role = "admin"
	withSetupQueryUser2(a, u1)
	gt := testSetRolesForTenantRequest(s.T(), t1, u1, MustJSONMarshal(
		TenantRolesRequest{
			Roles:          []string{"admin", "role1", "role1", "role2", "role1"},
			SubTenantRoles: []string{"admin", "guest", "srole1", "srole1", "srole2", "srole1"}}))
	withSetupQueryUsers(a, t1, []*multi.User{u1})

	expected1 := t1
	expected1.Roles = []string{"admin", "guest", "role1", "role2"}
	expected1.SubTenantRoles = []string{"admin", "guest", "srole1", "srole2"}
	av, _ := attributevalue.MarshalMap(t1)
	av["pk"] = &ddbtypes.AttributeValueMemberS{Value: db.GlobalEntityID("*")}
	av["sk"] = &ddbtypes.AttributeValueMemberS{Value: db.TenantEntityID(t1.ID.String())}
	av["altName"] = &ddbtypes.AttributeValueMemberS{Value: db.TenantEntityID(t1.Name)}
	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(a.DBCtx().TableName()),
	}
	a.ddb.EXPECT().PutItem(mock.Anything, input).Return(nil, nil)
	a.cogIdp.EXPECT().CreateGroup(mock.Anything, mock.Anything).Return(nil, nil)
	err := handleUpdateRolesForTenant(a, gt.c)
	s.NoError(err)
}
