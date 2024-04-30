package session

import (
	"context"
	"encoding/json"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"

	"go.uber.org/zap"
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
	ctx          context.Context
	aws          aws.Config
	LastError    error  `json:"last_error"`
	Config       Config `json:"config"`
	UserPool     string `json:"user_pool"`
	Email        string `json:"email"`
	AccessToken  string `json:"access_token"`
	IdToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
}

var theSession *Session

func Must() *Session {
	if theSession != nil {
		return theSession
	}
	var err error
	theSession, err = loadSessionFromFile()
	if err != nil {
		zap.L().Fatal("failed to load session from file", zap.Error(err))
	}
	return theSession
}

func newEmptySession() (*Session, error) {
	ctx := context.Background()
	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &Session{
		ctx:    context.Background(),
		aws:    awsCfg,
		Config: NewDefaultConfig(),
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

func (s *Session) Ctx() context.Context {
	return s.ctx
}

func (s *Session) AWS() aws.Config {
	return s.aws
}

func (s *Session) Logout() error {
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

func (s *Session) SetLastError(err error) {
	s.LastError = err
}

func (s *Session) ClearLastError() {
	s.LastError = nil
}

func (s *Session) GetLastError() error {
	return s.LastError
}
