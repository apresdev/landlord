package server

import (
	"net/url"

	"github.com/gin-gonic/gin"
)

const (
	LandlordLastError              = "landlord-last-error"
	LandlordUsernameCookie         = "landlord-username"
	LandlordChallengeSessionCookie = "landlord-challenge-session"
	LandlordIDTokenCookie          = "landlord-id-token"
	LandlordAccessTokenCookie      = "landlord-access-token"
	LandlordRefreshTokenCookie     = "landlord-refresh-token"
	LandlordCookieMaxAge           = 3600
)

func isSecure(u *url.URL) bool {
	return u.Scheme == "https"
}

func setCookie(c *gin.Context, u *url.URL, cookie, value string) {
	c.SetCookie(
		cookie,
		value,
		LandlordCookieMaxAge,
		"/",
		u.Hostname(),
		isSecure(u),
		true)
}

func clearCookie(c *gin.Context, u *url.URL, cookie string) {
	c.SetCookie(
		cookie,
		"",
		-1,
		"/",
		u.Hostname(),
		isSecure(u),
		true)
}

func clearCookies(c *gin.Context, u *url.URL, cookies ...string) {
	for _, cookie := range cookies {
		clearCookie(c, u, cookie)
	}
}

func clearLandlordCookies(c *gin.Context, u *url.URL) {
	clearCookies(c, u,
		LandlordUsernameCookie,
		LandlordChallengeSessionCookie,
		LandlordIDTokenCookie,
		LandlordAccessTokenCookie,
		LandlordRefreshTokenCookie)
}
