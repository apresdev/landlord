package cmd_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/apresdev/landlord/internal/visit"

	"github.com/apresdev/landlord/internal/localstack"

	"github.com/apresdev/landlord/cmd"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

type TenantTestSuite struct {
	suite.Suite
	out    bytes.Buffer
	errOut bytes.Buffer
	cmd    *cobra.Command
}

func TestTenantTestSuite(t *testing.T) {
	if !localstack.IsRunning() {
		t.Skip("localstack not running")
	}
	suite.Run(t, &TenantTestSuite{})
}

func (s *TenantTestSuite) SetupTest() {
	s.cmd = cmd.Root()
	s.cmd.SetOut(&s.out)
	s.cmd.SetErr(&s.errOut)
	err := visit.FilesInDirectory(".", []string{".json"}, func(filename string) error {
		if strings.HasPrefix(filename, "tenants.") {
			return os.Remove(filename)
		}
		return nil
	})
	s.Require().NoError(err)
	cmd.ClearSession()
}

func (s *TenantTestSuite) TearDownTest() {
	s.out.Reset()
}

func (s *TenantTestSuite) CLI(args string) (*cobra.Command, error) {
	s.cmd.SetArgs(strings.Fields(args))
	return s.cmd.ExecuteC()
}

func (s *TenantTestSuite) TestCreateTenant() {
	c, err := s.CLI("tenant create --name ten1")
	s.NoError(err)
	s.Equal("create", c.Use)
	s.Equal("Created tenant: ten1\n", s.out.String())
}
