package webapp

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/apresdev/landlord/internal/apperr"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/securecookie"
	"go.uber.org/zap"
)

type CookieName string

const (
	LandlordUsernameCookie         CookieName = "landlord-username"
	LandlordChallengeSessionCookie CookieName = "landlord-challenge-session"
	LandlordAccessTokenCookie      CookieName = "landlord-access-token"
	LandlordRefreshTokenCookie     CookieName = "landlord-refresh-token"
	LandlordOTPURICookie           CookieName = "landlord-otp-uri"
	LandlordCookieMaxAge                      = 3600
)

func isSecure(u *url.URL) bool {
	return u.Scheme == "https"
}

func clearCookie(c *gin.Context, u *url.URL, cookie CookieName) {
	c.SetCookie(
		string(cookie),
		"",
		-1,
		"/",
		u.Hostname(),
		isSecure(u),
		true)
}

func clearCookies(c *gin.Context, u *url.URL, cookies ...CookieName) {
	for _, cookie := range cookies {
		clearCookie(c, u, cookie)
	}
}

func clearLandlordCookies(c *gin.Context, u *url.URL) {
	clearCookies(c, u,
		LandlordUsernameCookie,
		LandlordChallengeSessionCookie,
		LandlordAccessTokenCookie,
		LandlordRefreshTokenCookie,
		LandlordOTPURICookie)
}

var sCookie *securecookie.SecureCookie

// InitSecureCookies initializes the secure cookie handler (sCookie).
// This function must be called before any cookie-related operations
// to ensure that sCookie is properly initialized. Failure to do so
// will result in a fatal error.
func InitSecureCookies(a App) {
	rawCookieKey := os.Getenv("LANDLORD_SECURE_COOKIE_KEY")
	if rawCookieKey == "" {
		zap.L().Fatal("LANDLORD_SECURE_COOKIE_KEY is not set")
	}
	cookieKey, err := base64.StdEncoding.DecodeString(rawCookieKey)
	if err != nil {
		zap.L().Fatal("failed to decode LANDLORD_SECURE_COOKIE_KEY",
			zap.String("raw_cookie_key", rawCookieKey),
			zap.Error(err))
	}
	if len(cookieKey) != 64 {
		zap.L().Fatal("LANDLORD_SECURE_COOKIE_KEY must be 64 bytes long",
			zap.Int("length", len(cookieKey)),
			zap.String("raw_cookie_key", rawCookieKey))
	}
	hashKey := cookieKey[:32]
	blockKey := cookieKey[32:]
	sCookie = securecookie.New(hashKey, blockKey)
}

func setEncodedCookie(a App, c *gin.Context, name CookieName, value string) {
	if sCookie == nil {
		zap.L().Fatal("secure cookie not initialized")
	}

	encodedValue, err := sCookie.Encode(string(name), value)
	if err != nil {
		zap.L().Error("failed to encode cookie",
			zap.String("name", string(name)),
			zap.String("value", value),
			zap.Error(err))
		return
	}
	encodedCookie := &http.Cookie{
		Name:     string(name),
		Value:    encodedValue,
		Path:     "/",
		MaxAge:   LandlordCookieMaxAge,
		HttpOnly: true,
		Secure:   !a.Settings().DevMode(),
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(c.Writer, encodedCookie)
}

func getEncodedCookie(c *gin.Context, name CookieName) (string, error) {
	if sCookie == nil {
		zap.L().Fatal("secure cookie not initialized")
	}

	cookieValue, err := c.Cookie(string(name))
	if err != nil {
		switch {
		case errors.Is(err, http.ErrNoCookie):
			return "", apperr.NewUnauthorized("unauthorized")
		default:
			return "", apperr.NewInternalE(fmt.Errorf("failed to get cookie: %w", err))
		}
	}

	var decodedValue string
	err = sCookie.Decode(string(name), cookieValue, &decodedValue)
	if err != nil {
		zap.L().Error("failed to decode cookie",
			zap.String("cid", string(clientID(c))),
			zap.String("name", string(name)),
			zap.String("value", cookieValue),
			zap.Error(err))
		return "", apperr.NewUnauthorized("unauthorized")
	}

	return decodedValue, nil
}
