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

type RootTestSuite struct {
	suite.Suite
	s   appenv.Settings
	out bytes.Buffer
	cmd *cobra.Command
}

func TestRootTestSuite(t *testing.T) {
	suite.Run(t, &RootTestSuite{s: appenv.MustTestSettings()})
}

func (s *RootTestSuite) SetupSuite() {
	if !apptest.InitLocalStack(s.s) {
		s.T().Skip("localstack not running")
	}
}

func (s *RootTestSuite) SetupTest() {
	s.cmd = cmd.Root()
	s.cmd.SetOut(&s.out)
	apptest.ResetLocalStack(s.s)
}

func (s *RootTestSuite) TearDownTest() {
	s.out.Reset()
}

func (s *RootTestSuite) CLI(args string) (*cobra.Command, error) {
	s.cmd.SetArgs(strings.Fields(args))
	return s.cmd.ExecuteC()
}

func (s *RootTestSuite) TestRoot() {
	s.cmd.SetArgs([]string{"-h"})
	c, err := s.cmd.ExecuteC()
	s.NoError(err)
	s.Equal("landlord", c.Use)
	s.Equal("Manage tenants in AWS Cognito", c.Short)
	s.True(s.out.Len() > 0)
	s.NotEqual(strings.Index(s.out.String(), "Usage:"), -1)
}

func (s *RootTestSuite) TestLogoutAndVerify() {
	_, err := s.CLI("logout")
	s.NoError(err)
	ses := cmd.MustReloadSession()
	s.Empty(ses.UserPool)
	s.Empty(ses.ValidationURL)
	s.Empty(ses.AccessToken)
	s.Empty(ses.IdToken)
	s.Empty(ses.RefreshToken)
}
