package cmd

import (
	"encoding/json"
	"os"

	"github.com/apresdev/landlord/internal/authx"

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
	LastError    error  `json:"last_error"`
	Config       Config `json:"config"`
	UserPool     string `json:"user_pool"`
	Email        string `json:"email"`
	AccessToken  string `json:"access_token"`
	IdToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
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

func ClearSession() {
	theSession = nil
}

func MustReloadSession() *Session {
	theSession = nil
	return MustSession()
}

func newEmptySession() (*Session, error) {
	return &Session{
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

func MustTenants(cs Clients) *authx.Tenants {
	tens, err := authx.LoadTenants(cs.Ctx(), cs.Cfg(), cs.Sts(), cs.CogIdp())
	if err != nil {
		panic(err)
	}
	return tens
}
