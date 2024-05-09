package cmd_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/apresdev/landlord/internal/localstack"

	"github.com/apresdev/landlord/cmd"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

type ConfirmTestSuite struct {
	suite.Suite
	out    bytes.Buffer
	errOut bytes.Buffer
	cmd    *cobra.Command
}

func TestConfirmTestSuite(t *testing.T) {
	if !localstack.IsRunning() {
		t.Skip("localstack not running")
	}
	suite.Run(t, &ConfirmTestSuite{})
}

func (s *ConfirmTestSuite) SetupTest() {
	s.cmd = cmd.Root()
	s.cmd.SetOut(&s.out)
	s.cmd.SetErr(&s.errOut)
}

func (s *ConfirmTestSuite) TearDownTest() {
	s.out.Reset()
}

func (s *ConfirmTestSuite) CLI(args string) (*cobra.Command, error) {
	s.cmd.SetArgs(strings.Fields(args))
	return s.cmd.ExecuteC()
}

func (s *ConfirmTestSuite) TestSimple() {
	c, err := s.CLI("confirm --userpool not-a-user-pool1 --email test-email1 --code test-code1")
	s.Error(err)
	s.Equal("confirm", c.Use)
	s.Contains(err.Error(), "no user pool found")
}

func (s *ConfirmTestSuite) TestMissingUserPool() {
	c, err := s.CLI("confirm --email test-email2 --code test-code2")
	s.Error(err)
	s.Equal(c.Use, "confirm")
	s.Contains(err.Error(), "--userpool required")
}

func (s *ConfirmTestSuite) TestMissingEmail() {
	c, err := s.CLI("confirm --userpool not-a-user-pool3 --code test-code3")
	s.Error(err)
	s.Equal(c.Use, "confirm")
	s.Contains(err.Error(), "--email required")
}

func (s *ConfirmTestSuite) TestMissingCode() {
	c, err := s.CLI("confirm --userpool not-a-user-pool4 --email test-email4")
	s.Error(err)
	s.Equal(c.Use, "confirm")
	s.Contains(err.Error(), "--code required")
}
