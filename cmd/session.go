package cmd

import (
	"context"
	"encoding/json"
	"net/url"
	"os"

	"github.com/apresdev/landlord/internal/appcore"
	"github.com/apresdev/landlord/internal/appenv"
	"go.uber.org/zap"
)

type Session struct {
	ValidationURL    string `json:"validation_url"`
	UserPool         string `json:"user_pool"`
	AccessToken      string `json:"access_token"`
	IdToken          string `json:"id_token"`
	RefreshToken     string `json:"refresh_token"`
	ChallengeSession string `json:"challenge_session"`
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
	return &Session{}, nil
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
	s.ValidationURL = ""
	s.UserPool = ""
	s.AccessToken = ""
	s.IdToken = ""
	s.RefreshToken = ""
	s.ChallengeSession = ""
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

func (s *Session) GetValidationURL() *url.URL {
	u, err := url.Parse(s.ValidationURL)
	if err != nil {
		zap.L().Fatal("could not parse URL", zap.Error(err))
	}
	return u
}

type SettingsKeyType string

const SettingsKey SettingsKeyType = "settings"

func appcoreMust(ctx context.Context) appcore.Clients {
	s := ctx.Value(SettingsKey)
	if s == nil {
		return appcore.Must()
	}
	settings, ok := s.(appenv.Settings)
	if !ok {
		zap.L().Fatal("invalid settings type", zap.Any("type", s))
	}
	return appcore.MustWithSettings(settings)
}
