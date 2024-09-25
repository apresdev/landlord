package cmd_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/apresdev/landlord/internal/apptest"

	"github.com/apresdev/landlord/internal/localstack"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/mock"

	"github.com/apresdev/landlord/cmd"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

type InfoTestSuite struct {
	suite.Suite
	out    bytes.Buffer
	errOut bytes.Buffer
	cmd    *cobra.Command
}

func TestInfoTestSuite(t *testing.T) {
	if !localstack.IsRunning() {
		t.Skip("localstack not running")
	}
	suite.Run(t, &InfoTestSuite{})
}

func (s *InfoTestSuite) SetupTest() {
	s.cmd = cmd.Root()
	s.cmd.SetOut(&s.out)
	s.cmd.SetErr(&s.errOut)
	ses := cmd.MustSession()

	ses.UserPool = "test-user-pool"
	ses.ValidationURL = "https://example.com/test-validation-url"
	ses.AccessToken = "test-access-token"
	ses.IdToken = "test-id-token"
	ses.RefreshToken = "test-refresh-token"
}

func (s *InfoTestSuite) TearDownTest() {
	s.out.Reset()
}

func (s *InfoTestSuite) CLI(args string) (*cobra.Command, error) {
	s.cmd.SetArgs(strings.Fields(args))
	return s.cmd.ExecuteC()
}

func (s *InfoTestSuite) TestSimple() {
	c, err := s.CLI("info")
	s.NoError(err)
	s.Equal("info", c.Use)
	f := strings.Split(s.out.String(), "\n")
	s.Equal("User pool : test-user-pool", f[0])
	s.Equal("Username     : test-email", f[1])
	s.Equal("Tokens:", f[2])
	s.Equal("  Access  : valid", f[3])
	s.Equal("  ID      : valid", f[4])
	s.Equal("  Refresh : valid", f[5])
}

func (s *InfoTestSuite) TestVerboseInValid() {
	m, reset := apptest.MockClientsOverride(s.T())
	defer reset()
	m.MockCogIdp.EXPECT().DescribeUserPool(m.Ctx(), mock.Anything).Return(
		&cognitoidentityprovider.DescribeUserPoolOutput{
			UserPool: &types.UserPoolType{
				Id: aws.String("test-user-pool"),
			}}, nil).Once()
	m.MockValidator.EXPECT().ValidateToken(
		m.Ctx(), "mock-region", mock.Anything, "test-access-token").Return(
		&jwt.Token{
			Raw:       "",
			Method:    nil,
			Header:    nil,
			Claims:    nil,
			Signature: nil,
			Valid:     false,
		}, nil)
	m.MockValidator.EXPECT().ValidateToken(
		m.Ctx(), "mock-region", mock.Anything, "test-id-token").Return(
		&jwt.Token{
			Raw:       "",
			Method:    nil,
			Header:    nil,
			Claims:    nil,
			Signature: nil,
			Valid:     false,
		}, nil)
	c, err := s.CLI("info --verbose")
	s.NoError(err)
	s.Equal("info", c.Use)
	expected := `User pool : test-user-pool
Username     : test-email
Tokens:
  Access:
    Valid : false
    Header: empty
  ID:
    Valid : false
    Header: empty
`
	s.Equal(expected, s.out.String())
}

func (s *InfoTestSuite) TestVerboseValid() {
	m, reset := apptest.MockClientsOverride(s.T())
	defer reset()
	m.MockCogIdp.EXPECT().DescribeUserPool(m.Ctx(), mock.Anything).Return(
		&cognitoidentityprovider.DescribeUserPoolOutput{
			UserPool: &types.UserPoolType{
				Id: aws.String("test-user-pool"),
			}}, nil).Once()
	m.MockValidator.EXPECT().ValidateToken(
		m.Ctx(), m.Cfg().Region, mock.Anything, "test-access-token").Return(
		&jwt.Token{
			Raw:    "",
			Method: jwt.GetSigningMethod(jwt.SigningMethodHS512.Alg()),
			Header: map[string]interface{}{"alg": jwt.SigningMethodHS512.Alg()},
			Claims: jwt.RegisteredClaims{
				Issuer:    "test-access-token-issuer",
				Subject:   "test-access-token-subject",
				Audience:  []string{"test-access-token-audience"},
				ExpiresAt: &jwt.NumericDate{},
				NotBefore: &jwt.NumericDate{},
				IssuedAt:  &jwt.NumericDate{},
				ID:        "test-access-token-id",
			},
			Signature: nil,
			Valid:     false,
		}, nil)
	m.MockValidator.EXPECT().ValidateToken(
		m.Ctx(), m.Cfg().Region, mock.Anything, "test-id-token").Return(
		&jwt.Token{
			Raw:    "",
			Method: jwt.GetSigningMethod(jwt.SigningMethodHS512.Alg()),
			Header: map[string]interface{}{"alg": jwt.SigningMethodHS512.Alg()},
			Claims: jwt.RegisteredClaims{
				Issuer:    "test-id-token-issuer",
				Subject:   "test-id-token-subject",
				Audience:  []string{"test-id-token-audience"},
				ExpiresAt: &jwt.NumericDate{},
				NotBefore: &jwt.NumericDate{},
				IssuedAt:  &jwt.NumericDate{},
				ID:        "test-id-token-id",
			},
			Signature: nil,
			Valid:     false,
		}, nil)
	c, err := s.CLI("info --verbose")
	s.NoError(err)
	s.Equal("info", c.Use)
	expected := `User pool : test-user-pool
Username     : test-email
Tokens:
  Access:
    Valid : false
    Method: HS512
    Header:
      alg: HS512
    Issuer: test-access-token-issuer
    Subject: test-access-token-subject
  ID:
    Valid : false
    Method: HS512
    Header:
      alg: HS512
    Issuer: test-id-token-issuer
    Subject: test-id-token-subject
`
	s.Equal(expected, s.out.String())
}

func (s *InfoTestSuite) TestVerboseRaw() {
	m, reset := apptest.MockClientsOverride(s.T())
	defer reset()
	m.MockCogIdp.EXPECT().DescribeUserPool(m.Ctx(), mock.Anything).Return(
		&cognitoidentityprovider.DescribeUserPoolOutput{
			UserPool: &types.UserPoolType{
				Id: aws.String("test-user-pool"),
			}}, nil).Once()
	m.MockValidator.EXPECT().ValidateToken(
		m.Ctx(), m.Cfg().Region, mock.Anything, "test-access-token").Return(
		&jwt.Token{
			Raw:    "raw-access-token",
			Method: jwt.GetSigningMethod(jwt.SigningMethodHS512.Alg()),
			Header: map[string]interface{}{"alg": jwt.SigningMethodHS512.Alg()},
			Claims: jwt.RegisteredClaims{
				Issuer:    "test-access-token-issuer",
				Subject:   "test-access-token-subject",
				Audience:  []string{"test-access-token-audience"},
				ExpiresAt: &jwt.NumericDate{},
				NotBefore: &jwt.NumericDate{},
				IssuedAt:  &jwt.NumericDate{},
				ID:        "test-access-token-id",
			},
			Signature: nil,
			Valid:     false,
		}, nil)
	m.MockValidator.EXPECT().ValidateToken(
		m.Ctx(), m.Cfg().Region, mock.Anything, "test-id-token").Return(
		&jwt.Token{
			Raw:    "raw-id-token",
			Method: jwt.GetSigningMethod(jwt.SigningMethodHS512.Alg()),
			Header: map[string]interface{}{"alg": jwt.SigningMethodHS512.Alg()},
			Claims: jwt.RegisteredClaims{
				Issuer:    "test-id-token-issuer",
				Subject:   "test-id-token-subject",
				Audience:  []string{"test-id-token-audience"},
				ExpiresAt: &jwt.NumericDate{},
				NotBefore: &jwt.NumericDate{},
				IssuedAt:  &jwt.NumericDate{},
				ID:        "test-id-token-id",
			},
			Signature: nil,
			Valid:     false,
		}, nil)
	c, err := s.CLI("info --verbose --raw")
	s.NoError(err)
	s.Equal("info", c.Use)
	expected := `User pool : test-user-pool
Username     : test-email
Tokens:
  Access:
    Valid : false
    Method: HS512
    Header:
      alg: HS512
    Issuer: test-access-token-issuer
    Subject: test-access-token-subject
  ID:
    Valid : false
    Method: HS512
    Header:
      alg: HS512
    Issuer: test-id-token-issuer
    Subject: test-id-token-subject
Access:
test-access-token
ID:
test-id-token
Refresh:
test-refresh-token
`
	s.Equal(expected, s.out.String())
}
