package rest

import (
	"bytes"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/apresdev/landlord/internal/perms"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"

	"github.com/apresdev/landlord/internal/appcore"
	"github.com/apresdev/landlord/internal/authx"
	"github.com/apresdev/landlord/internal/db"
	"github.com/apresdev/landlord/internal/multi"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/apresdev/landlord/internal/appenv"
	"github.com/apresdev/landlord/internal/apptest"

	"github.com/stretchr/testify/suite"
)

type HandleUpdateUserRoleTestSuite struct {
	suite.Suite
	ginMode string
	s       appenv.Settings
}

func TestHandleUpdateUserRoleTestSuite(t *testing.T) {
	suite.Run(t, &HandleUpdateUserRoleTestSuite{s: appenv.MustTestSettings()})
}

func (s *HandleUpdateUserRoleTestSuite) SetupSuite() {
	if !apptest.InitLocalStack(s.s) {
		s.T().Skip("localstack not running")
	}
}

func (s *HandleUpdateUserRoleTestSuite) TearDownSuite() {
	apptest.ResetLocalStack(s.s)
}

func (s *HandleUpdateUserRoleTestSuite) SetupTest() {
	s.ginMode = gin.Mode()
	gin.SetMode(gin.TestMode)
	apptest.ResetLocalStack(s.s)
}

func (s *HandleUpdateUserRoleTestSuite) TestHappyPath() {
	c := appcore.MustWithSettings(s.s)
	rootTenant, err := db.GetRootTenant(c.Ctx(), c.DBCtx(), c.RootTenantID())
	s.NoError(err)
	username := "user1@example.com"
	err = authx.SendInvite(
		c.Ctx(),
		c.DBCtx(),
		c.CogIdp(),
		rootTenant,
		username,
		uuid.Nil,
		"admin",
		multi.MFAPolicyNone)
	s.NoError(err)

	o, err := c.CogIdp().AdminListGroupsForUser(c.Ctx(),
		&cognitoidentityprovider.AdminListGroupsForUserInput{
			UserPoolId: aws.String(rootTenant.UserPoolID),
			Username:   aws.String(username),
		})
	s.NoError(err)
	s.Len(o.Groups, 1)
	s.Equal("tenant-"+rootTenant.ID.String()+"-admin", *o.Groups[0].GroupName)

	u, err := db.GetUserByName(c.Ctx(), c.DBCtx(), username)
	s.NoError(err)
	assert.Equal(s.T(), username, u.Username)
	assert.Equal(s.T(), "admin", u.Role)

	req := UpdateUserRoleRequest{Role: "guest"}
	reqURL := fmt.Sprintf("/ll-auth/api/tenants/%s/users/%s/role",
		rootTenant.ID.String(), u.ID.String())
	req2 := httptest.NewRequest(
		"POST", reqURL, bytes.NewBuffer(MustJSONMarshal(req)))
	gt := newGinTest(s.T()).withRequest(req2)
	gt.c.AddParam("tenantID", rootTenant.ID.String())
	gt.c.AddParam("userID", u.ID.String())

	claims := make(jwt.MapClaims)
	claims[authx.SubClaim] = u.CognitoID.String()
	claims[authx.CustomTenantClaim] = rootTenant.ID.String()
	claims[authx.CognitoGroupsClaim] = []string{
		fmt.Sprintf("tenant-%s-admin", rootTenant.ID.String())}
	claims[authx.EmailClaim] = u.Username
	tok := &jwt.Token{Claims: claims}
	gt.c.Set(perms.TokenKey, tok)
	err = handleUpdateUserRole(c, gt.c)
	s.NoError(err)

	o, err = c.CogIdp().AdminListGroupsForUser(c.Ctx(),
		&cognitoidentityprovider.AdminListGroupsForUserInput{
			UserPoolId: aws.String(rootTenant.UserPoolID),
			Username:   aws.String(username),
		})
	s.NoError(err)
	s.Len(o.Groups, 1)
	s.Equal("tenant-"+rootTenant.ID.String()+"-guest", *o.Groups[0].GroupName)
}
