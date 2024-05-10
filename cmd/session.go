package cmd

import (
	"context"
	"encoding/json"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/iam"

	"github.com/apresdev/landlord/internal/authx"

	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"

	"github.com/aws/aws-sdk-go-v2/service/cognitoidentity"

	"go.uber.org/zap"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

type Config struct {
	LogLimit int `json:"log_limit"`
}

func NewDefaultConfig() Config {
	return Config{
		LogLimit: 15,
	}
}

type Session struct {
	Ctx          context.Context                  `json:"-"`
	AWSConfig    aws.Config                       `json:"-"`
	Validator    authx.TokenValidator             `json:"-"`
	CogId        authx.AWSCognitoIdentity         `json:"-"`
	CogIdp       authx.AWSCognitoIdentityProvider `json:"-"`
	IAM          authx.AWSIAM                     `json:"-"`
	LastError    error                            `json:"last_error"`
	Config       Config                           `json:"config"`
	UserPool     string                           `json:"user_pool"`
	Email        string                           `json:"email"`
	AccessToken  string                           `json:"access_token"`
	IdToken      string                           `json:"id_token"`
	RefreshToken string                           `json:"refresh_token"`
}

var theSession *Session

func MustSession() *Session {
	if theSession != nil {
		return theSession
	}
	var err error
	theSession, err = loadSessionFromFile()
	if err != nil {
		theSession, err = newEmptySession()
		if err != nil {
			zap.L().Fatal("failed to create session", zap.Error(err))
		}
	}
	return theSession
}

func MustReloadSession() *Session {
	theSession = nil
	return MustSession()
}

func newEmptySession() (*Session, error) {
	ctx := context.Background()
	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &Session{
		Ctx:       context.Background(),
		AWSConfig: awsCfg,
		CogId:     cognitoidentity.NewFromConfig(awsCfg),
		CogIdp:    cognitoidentityprovider.NewFromConfig(awsCfg),
		IAM:       iam.NewFromConfig(awsCfg),
		Validator: authx.NewTokenValidator(),
		Config:    NewDefaultConfig(),
	}, nil
}

func loadSessionFromFile() (*Session, error) {
	s, err := newEmptySession()
	if err != nil {
		return nil, err
	}
	file, err := os.Open("session.json")
	if err != nil {
		return nil, err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)
	err = json.NewDecoder(file).Decode(&s)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Session) Logout() error {
	s.UserPool = ""
	s.Email = ""
	s.AccessToken = ""
	s.IdToken = ""
	s.RefreshToken = ""
	return s.SaveToFile()
}

func (s *Session) SaveToFile() error {
	file, err := os.Create("session.json")
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)
	enc := json.NewEncoder(file)
	enc.SetIndent("  ", "  ")
	err = enc.Encode(s)
	if err != nil {
		return err
	}
	return nil
}

func (s *Session) GetRegion() string {
	return s.AWSConfig.Region
}

func (s *Session) SetLastError(err error) {
	s.LastError = err
}

func (s *Session) ClearLastError() {
	s.LastError = nil
}

func (s *Session) GetLastError() error {
	return s.LastError
}

func (s *Session) StoreIdentity(userpool, email string) error {
	s.UserPool = userpool
	s.Email = email
	return s.SaveToFile()
}

func (s *Session) StoreTokens(accessToken, idToken, refreshToken string) error {
	s.AccessToken = accessToken
	s.IdToken = idToken
	s.RefreshToken = refreshToken
	return s.SaveToFile()
}
