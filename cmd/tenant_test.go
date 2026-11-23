package cmd_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/apresdev/landlord/internal/appenv"
	"github.com/apresdev/landlord/internal/apptest"

	"github.com/apresdev/landlord/cmd"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

type TenantTestSuite struct {
	suite.Suite
	s      appenv.Settings
	out    bytes.Buffer
	errOut bytes.Buffer
	ctx    context.Context
	cmd    *cobra.Command
}

func TestTenantTestSuite(t *testing.T) {
	suite.Run(t, &TenantTestSuite{s: appenv.MustTestSettings()})
}

func (s *TenantTestSuite) SetupSuite() {
	if !apptest.InitEmulator(s.s) {
		s.T().Skip("AWS emulator not running")
	}
}

func (s *TenantTestSuite) SetupTest() {
	s.cmd = cmd.Root()
	s.cmd.SetOut(&s.out)
	s.cmd.SetErr(&s.errOut)
	s.ctx = context.Background()
	s.ctx = context.WithValue(s.ctx, cmd.SettingsKey, s.s)
	cmd.ClearSession()
	apptest.ResetEmulator(s.s)
}

func (s *TenantTestSuite) TearDownTest() {
	s.out.Reset()
}

func (s *TenantTestSuite) CLI(args string) (*cobra.Command, error) {
	s.cmd.SetArgs(strings.Fields(args))
	return s.cmd.ExecuteContextC(s.ctx)
}

func (s *TenantTestSuite) TestCreateTenant() {
	c, err := s.CLI("tenant create --name ten1")
	s.NoError(err)
	s.Equal("create", c.Use)
	s.Equal("Created tenant: ten1\n", s.out.String())
}
