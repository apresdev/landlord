package cmd_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/apresdev/landlord/internal/appenv"
	"github.com/apresdev/landlord/internal/apptest"

	"github.com/apresdev/landlord/cmd"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

type LoginTestSuite struct {
	suite.Suite
	s      appenv.Settings
	out    bytes.Buffer
	errOut bytes.Buffer
	cmd    *cobra.Command
}

func TestLoginTestSuite(t *testing.T) {
	suite.Run(t, &LoginTestSuite{s: appenv.MustTestSettings()})
}

func (s *LoginTestSuite) SetupSuite() {
	if !apptest.InitEmulator(s.s) {
		s.T().Skip("AWS emulator not running")
	}
}

func (s *LoginTestSuite) SetupTest() {
	s.cmd = cmd.Root()
	s.cmd.SetOut(&s.out)
	s.cmd.SetErr(&s.errOut)
}

func (s *LoginTestSuite) TearDownTest() {
	s.out.Reset()
}

func (s *LoginTestSuite) CLI(args string) (*cobra.Command, error) {
	s.cmd.SetArgs(strings.Fields(args))
	return s.cmd.ExecuteC()
}

func (s *LoginTestSuite) TestMissingUserPool() {
	c, err := s.CLI("login --email test-email2 --password test-password2")
	s.Error(err)
	s.Equal(c.Use, "login")
	s.Contains(err.Error(), "--userpool required")
}

func (s *LoginTestSuite) TestMissingEmail() {
	c, err := s.CLI("login --userpool not-a-user-pool3 --password test-password3")
	s.Error(err)
	s.Equal(c.Use, "login")
	s.Contains(err.Error(), "--username required")
}

func (s *LoginTestSuite) TestMissingPassword() {
	c, err := s.CLI("login --userpool not-a-user-pool4 --email test-email4")
	s.Error(err)
	s.Equal(c.Use, "login")
	s.Contains(err.Error(), "--password required")
}
