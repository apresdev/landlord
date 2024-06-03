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

type SignUpTestSuite struct {
	suite.Suite
	out    bytes.Buffer
	errOut bytes.Buffer
	cmd    *cobra.Command
}

func TestSignUpTestSuite(t *testing.T) {
	if !localstack.IsRunning() {
		t.Skip("localstack not running")
	}
	suite.Run(t, &SignUpTestSuite{})
}

func (s *SignUpTestSuite) SetupTest() {
	s.cmd = cmd.Root()
	s.cmd.SetOut(&s.out)
	s.cmd.SetErr(&s.errOut)
}

func (s *SignUpTestSuite) TearDownTest() {
	s.out.Reset()
}

func (s *SignUpTestSuite) CLI(args string) (*cobra.Command, error) {
	s.cmd.SetArgs(strings.Fields(args))
	return s.cmd.ExecuteC()
}

func (s *SignUpTestSuite) TestSimple() {
	c, err := s.CLI("signup confirm --userpool not-a-user-pool1 --email test-email1 --code test-code1")
	s.Error(err)
	s.Equal("confirm", c.Use)
	s.Contains(err.Error(), "no user pool found")
}

func (s *SignUpTestSuite) TestMissingUserPool() {
	c, err := s.CLI("signup confirm --email test-email2 --code test-code2")
	s.Error(err)
	s.Equal(c.Use, "confirm")
	s.Contains(err.Error(), "--userpool required")
}

func (s *SignUpTestSuite) TestMissingEmail() {
	c, err := s.CLI("signup confirm --userpool not-a-user-pool3 --code test-code3")
	s.Error(err)
	s.Equal(c.Use, "confirm")
	s.Contains(err.Error(), "--email required")
}

func (s *SignUpTestSuite) TestMissingCode() {
	c, err := s.CLI("signup confirm --userpool not-a-user-pool4 --email test-email4")
	s.Error(err)
	s.Equal(c.Use, "confirm")
	s.Contains(err.Error(), "--code or --admin required")
}
