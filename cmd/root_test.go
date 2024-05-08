package cmd_test

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/apresdev/landlord/internal/localstack"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/google/uuid"

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
	if !localstack.IsRunning() {
		t.Skip(" localstack not running")
	}
	suite.Run(t, &RootTestSuite{})
}

func (s *RootTestSuite) SetupTest() {
	s.cmd = cmd.Root()
	s.cmd.SetOut(&s.out)
	ses := cmd.MustReloadSession()
	err := authx.DeleteAllUserPools(ses.Ctx, ses.CogIdp)
	s.NoError(err)
}

func (s *RootTestSuite) TearDownTest() {
	s.out.Reset()
	ses := cmd.MustReloadSession()
	err := authx.DeleteAllUserPools(ses.Ctx, ses.CogIdp)
	s.NoError(err)
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

func captureLocalstackLogs() (string, error) {
	c := exec.Command("localstack", "logs", "--tail", "20")
	var out bytes.Buffer
	c.Stdout = &out
	err := c.Run()
	if err != nil {
		return "", err
	}
	return out.String(), nil
}

func findConfirmationCodeInLogs(logs, email string) string {
	pattern := `Confirmation code for Cognito user\s+[^:]+:\s*(\d+)$`
	re := regexp.MustCompile(pattern)
	for _, line := range strings.Split(logs, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, email) {
			match := re.FindStringSubmatch(line)
			if match != nil {
				return match[1]
			}
		}
	}
	return ""
}

func (s *RootTestSuite) TestSignUpScenario() {
	testRunUUID := uuid.New().String()
	userPoolName := fmt.Sprintf("test-userpool1-%s", testRunUUID)
	c, err := s.CLI(fmt.Sprintf("userpool create --name %s", userPoolName))
	s.NoError(err)
	s.Equal("create", c.Use)
	s.Contains(s.out.String(), "Created user pool")
	testEmail := fmt.Sprintf("%s@example.com", testRunUUID)
	c, err = s.CLI(fmt.Sprintf(
		"signup --userpool %s --email %s --password Test-password1",
		userPoolName, testEmail))
	s.NoError(err)
	s.Equal("signup", c.Use)
	logs, err := captureLocalstackLogs()
	s.NoError(err)
	code := findConfirmationCodeInLogs(logs, testEmail)
	s.NotEmpty(code)
	c, err = s.CLI(fmt.Sprintf(
		"confirm --userpool %s --email %s --code %s",
		userPoolName, testEmail, code))
	s.NoError(err)
	c, err = s.CLI(fmt.Sprintf(
		"login --userpool %s --email %s --password Test-password1",
		userPoolName, testEmail))
	s.NoError(err)
	ses := cmd.MustReloadSession()
	s.Equal(userPoolName, ses.UserPool)
	s.Equal(testEmail, ses.Email)
	s.NotEmpty(ses.AccessToken)
	s.NotEmpty(ses.IdToken)
	s.NotEmpty(ses.RefreshToken)
	c, err = s.CLI("logout")
	s.NoError(err)
	ses = cmd.MustReloadSession()
	s.Empty(ses.UserPool)
	s.Empty(ses.Email)
	s.Empty(ses.AccessToken)
	s.Empty(ses.IdToken)
	s.Empty(ses.RefreshToken)
}
