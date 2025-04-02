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
	out bytes.Buffer
	cmd *cobra.Command
}

func TestRootTestSuite(t *testing.T) {
	if !apptest.InitLocalStack(appenv.MustSettings()) {
		t.Skip("localstack not running")
	}
	suite.Run(t, &RootTestSuite{})
}

func (s *RootTestSuite) SetupTest() {
	s.cmd = cmd.Root()
	s.cmd.SetOut(&s.out)
	apptest.ResetLocalStack(appenv.MustSettings())
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
