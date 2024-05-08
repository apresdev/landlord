package cmd_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/apresdev/landlord/internal/authx"

	"github.com/apresdev/landlord/internal/localstack"

	"github.com/apresdev/landlord/cmd"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

type IDPoolTestSuite struct {
	suite.Suite
	out    bytes.Buffer
	errOut bytes.Buffer
	cmd    *cobra.Command
}

func TestIDPoolTestSuite(t *testing.T) {
	if !localstack.IsRunning() {
		t.Skip(" localstack not running")
	}
	suite.Run(t, &IDPoolTestSuite{})
}

func (s *IDPoolTestSuite) SetupTest() {
	s.cmd = cmd.Root()
	s.cmd.SetOut(&s.out)
	s.cmd.SetErr(&s.errOut)
	ses := cmd.MustReloadSession()
	err := authx.DeleteAllIdentityPools(ses.Ctx, ses.CogId)
	s.NoError(err)
}

func (s *IDPoolTestSuite) TearDownTest() {
	s.out.Reset()
	ses := cmd.MustReloadSession()
	err := authx.DeleteAllIdentityPools(ses.Ctx, ses.CogId)
	s.NoError(err)
}

func (s *IDPoolTestSuite) CLI(args string) (*cobra.Command, error) {
	s.cmd.SetArgs(strings.Fields(args))
	return s.cmd.ExecuteC()
}

func (s *IDPoolTestSuite) TestHelp() {
	c, err := s.CLI("idpool")
	s.NoError(err)
	s.Equal("idpool", c.Use)
	s.Contains(s.out.String(), "Identity pool sub-command")
}

func (s *IDPoolTestSuite) TestEmptyList() {
	c, err := s.CLI("idpool list")
	s.NoError(err)
	s.Equal("list", c.Use)
	s.Assert().Equal(
		"No identity pools found", strings.TrimSpace(s.out.String()))
}

func (s *IDPoolTestSuite) TestList() {
	ses := cmd.MustSession()
	err := authx.CreateIdentityPool(ses.Ctx, ses.CogId, "test-idpool1")
	s.NoError(err)
	err = authx.CreateIdentityPool(ses.Ctx, ses.CogId, "test-idpool2")
	s.NoError(err)

	c, err := s.CLI("idpool list")
	s.NoError(err)
	s.Equal("list", c.Use)
	s.Assert().Contains(s.out.String(), "test-idpool1")
	s.Assert().Contains(s.out.String(), "test-idpool2")
}
