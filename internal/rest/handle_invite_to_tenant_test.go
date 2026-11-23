package rest

import (
	"bytes"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/apresdev/landlord/internal/multi"

	"github.com/apresdev/landlord/internal/appcore"
	"github.com/apresdev/landlord/internal/appenv"
	"github.com/apresdev/landlord/internal/apptest"
	"github.com/apresdev/landlord/internal/db"
	"github.com/gin-gonic/gin"

	"github.com/stretchr/testify/suite"
)

type HandleInviteToTenantTestSuite struct {
	suite.Suite
	ginMode string
	s       appenv.Settings
}

func TestHandleInviteToTenantTestSuite(t *testing.T) {
	suite.Run(t, &HandleInviteToTenantTestSuite{s: appenv.MustTestSettings()})
}

func (s *HandleInviteToTenantTestSuite) SetupSuite() {
	if !apptest.InitEmulator(s.s) {
		s.T().Skip("AWS emulator not running")
	}
}

func (s *HandleInviteToTenantTestSuite) TearDownSuite() {
	apptest.ResetEmulator(s.s)
}

func (s *HandleInviteToTenantTestSuite) SetupTest() {
	s.ginMode = gin.Mode()
	gin.SetMode(gin.TestMode)
	apptest.ResetEmulator(s.s)
}

func (s *HandleInviteToTenantTestSuite) TestHappyPath() {
	c := appcore.MustWithSettings(s.s)
	rootTenant, err := db.GetRootTenant(c.Ctx(), c.DBCtx(), c.RootTenantID())
	s.NoError(err)
	username := "user1@example.com"
	req := InviteRequest{
		Username:  username,
		Role:      "admin",
		MfaPolicy: "none",
	}
	reqURL := fmt.Sprintf(
		"/ll-auth/api/tenants/%s/invite", rootTenant.ID.String())
	req2 := httptest.NewRequest(
		"POST", reqURL, bytes.NewBuffer(MustJSONMarshal(req)))
	gt := newGinTest(s.T()).withRequest(req2)
	gt.c.AddParam("tenantID", rootTenant.ID.String())

	rootUsername := "root@example.com"
	err = inviteRootUser(s.s, rootUsername)
	s.NoError(err)
	rootUser, err := db.GetUserByName(c.Ctx(), c.DBCtx(), rootUsername)
	s.NoError(err)
	withSetupJWT(gt.c, rootTenant, rootUser)

	err = handleInviteToTenant(c, gt.c)
	s.NoError(err)

	newUser, err := db.GetUserByName(c.Ctx(), c.DBCtx(), username)
	s.NoError(err)
	s.Equal(username, newUser.Username)

	var resp InviteResponse
	MustJSONUnmarshal(gt.w.Body.Bytes(), &resp)
	s.Equal(newUser.ID.String(), resp.InvitedUser.UserID)
	s.Equal(newUser.Username, resp.InvitedUser.Username)
	s.Equal(newUser.Role, resp.InvitedUser.Role)
	s.Equal(newUser.Status, multi.UserStatusInvited)
	s.Equal(newUser.CognitoID.String(), resp.InvitedUser.CognitoID)
}
