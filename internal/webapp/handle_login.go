package webapp

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/apresdev/landlord/internal/db"
	"github.com/apresdev/landlord/internal/jwx"
	"github.com/skip2/go-qrcode"

	"github.com/apresdev/landlord/internal/rest"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func handleRoot(a App, c *gin.Context) error {
	ua := c.GetHeader("User-Agent")
	if strings.Contains(strings.ToLower(ua), "healthcheck") {
		c.JSON(http.StatusNoContent, nil)
		return nil
	}

	username, _ := c.Cookie(LandlordUsernameCookie)
	if username == "" {
		redirectToLogin(c)
		return nil
	}

	var rawTokens jwx.RawTokens
	rawTokens.IdToken, _ = c.Cookie(LandlordIDTokenCookie)
	rawTokens.AccessToken, _ = c.Cookie(LandlordAccessTokenCookie)
	rawTokens.RefreshToken, _ = c.Cookie(LandlordRefreshTokenCookie)
	if rawTokens.IdToken == "" || rawTokens.AccessToken == "" || rawTokens.RefreshToken == "" {
		redirectToLogin(c)
		return nil
	}

	_, tenant, err := db.GetUserAndTenant(c.Request.Context(), a.DDB(), username)
	if err != nil {
		zap.L().Warn("error finding tenant for username",
			zap.String("username", username),
			zap.Error(err))
		clearLandlordCookies(c, a.Settings().ConsoleURL())
		redirectToLogin(c)
		return nil
	}
	validationURL := jwx.ValidationURL(a.Cfg().Region, tenant.UserPoolID)
	_, err = jwx.ValidateAllTokens(
		c.Request.Context(), clientID(c), a.Validator(), validationURL, rawTokens)
	if err != nil {
		redirectToLogin(c)
		return err
	}

	redirectTo(c, "admin/dashboard")

	return nil
}

func handleLogin(a App, c *gin.Context) error {
	username, _ := c.Cookie(LandlordUsernameCookie)
	uv := NewUnauthenticatedView(a.Settings()).WithPotentialUsername(username)
	return executeUnauthenticatedView(c, "loginPage", uv)
}

func handleLoginSubmit(a App, c *gin.Context) error {
	var req rest.LoginRequest
	req.Username = c.PostForm("username")
	req.Password = c.PostForm("password")
	resp, err := rest.HandleLogin(a, c, req)
	if err != nil {
		uv := NewUnauthenticatedView(a.Settings()).
			WithPotentialUsername(req.Username).WithErrMsg(err.Error())
		return executeUnauthenticatedView(c, "loginPage", uv)
	}
	switch resp.Status {
	case rest.LoginStatusSuccess:
		setCookie(c, a.Settings().ConsoleURL(), LandlordUsernameCookie, req.Username)
		setCookie(c, a.Settings().ConsoleURL(), LandlordAccessTokenCookie, resp.AccessToken)
		setCookie(c, a.Settings().ConsoleURL(), LandlordIDTokenCookie, resp.IdToken)
		setCookie(c, a.Settings().ConsoleURL(), LandlordRefreshTokenCookie, resp.RefreshToken)
		redirectToAppRoot(c)
		return nil
	case rest.LoginStatusNewPasswordRequired:
		setCookie(c, a.Settings().ConsoleURL(), LandlordUsernameCookie, req.Username)
		setCookie(c, a.Settings().ConsoleURL(), LandlordChallengeSessionCookie, resp.Session)
		redirectTo(c, "new-password")
		return nil
	case rest.LoginStatusPasswordResetRequired:
		setCookie(c, a.Settings().ConsoleURL(), LandlordUsernameCookie, req.Username)
		setCookie(c, a.Settings().ConsoleURL(), LandlordChallengeSessionCookie, resp.Session)
		redirectTo(c, "reset-password")
		return nil
	case rest.LoginStatusMFASetupRequired:
		setCookie(c, a.Settings().ConsoleURL(), LandlordUsernameCookie, req.Username)
		setCookie(c, a.Settings().ConsoleURL(), LandlordAccessTokenCookie, resp.AccessToken)
		setCookie(c, a.Settings().ConsoleURL(), LandlordOTPURI, resp.OtpURI)
		redirectTo(c, "setup-otp")
		return nil
	case rest.LoginStatusMFACodeRequired:
		setCookie(c, a.Settings().ConsoleURL(), LandlordUsernameCookie, req.Username)
		setCookie(c, a.Settings().ConsoleURL(), LandlordChallengeSessionCookie, resp.Session)
		redirectTo(c, "otp")
		return nil
	default:
		zap.L().Error("unexpected login status value",
			zap.String("status", string(resp.Status)))
		uv := NewUnauthenticatedView(a.Settings()).
			WithPotentialUsername(req.Username).
			WithErrMsg("unexpected login status value")
		return executeUnauthenticatedView(c, "loginPage", uv)
	}
}

func handleLogoutSubmit(a App, c *gin.Context) error {
	clearLandlordCookies(c, a.Settings().ConsoleURL())
	redirectToLogin(c)
	return nil
}

func handleQRCode(a App, c *gin.Context) error {
	_ = a
	uri := c.Query("uri")
	if uri == "" {
		c.String(http.StatusBadRequest, "missing uri")
		return nil
	}
	b, err := base64.URLEncoding.DecodeString(uri)
	if err != nil {
		c.String(http.StatusBadRequest, "invalid uri")
		return nil
	}
	png, err := qrcode.Encode(string(b), qrcode.Medium, 256)
	if err != nil {
		return err
	}
	c.Data(http.StatusOK, "image/png", png)
	return nil
}

func handleSetupOTP(a App, c *gin.Context) error {
	otpURI, _ := c.Cookie(LandlordOTPURI)
	otpImageURI := appPath(fmt.Sprintf(
		"/qrcode?uri=%s", base64.URLEncoding.EncodeToString([]byte(otpURI))))
	username, _ := c.Cookie(LandlordUsernameCookie)
	uv := NewUnauthenticatedView(a.Settings()).
		WithPotentialUsername(username).WithOTPImageLink(otpImageURI)
	return executeUnauthenticatedView(c, "setupOTPPage", uv)
}

func handleSetupOTPSubmit(a App, c *gin.Context) error {
	var req rest.VerifyOtpAssociationRequest
	req.Username = c.PostForm("username")
	req.OtpCode = c.PostForm("code")
	req.AccessToken, _ = c.Cookie(LandlordAccessTokenCookie)
	if req.AccessToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing access token"})
		return nil
	}
	err := rest.HandleVerifyOTPAssociation(a, c, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil
	}
	// TODO: add refresh token to request and re-fetch tokens here
	redirectToAppRoot(c)
	return nil
}

func handleOTP(a App, c *gin.Context) error {
	username, _ := c.Cookie(LandlordUsernameCookie)
	uv := NewUnauthenticatedView(a.Settings()).WithPotentialUsername(username)
	return executeUnauthenticatedView(c, "otpCodePage", uv)
}

func handleOTPSubmit(a App, c *gin.Context) error {
	var req rest.ValidateMfaCodeRequest
	req.Username = c.PostForm("username")
	req.MfaCode = c.PostForm("code")
	req.Session, _ = c.Cookie(LandlordChallengeSessionCookie)
	resp, err := rest.HandleValidateMFACode(a, c, req)
	if err != nil {
		return err
	}
	switch resp.Status {
	case rest.LoginStatusSuccess:
		setCookie(c, a.Settings().ConsoleURL(), LandlordUsernameCookie, req.Username)
		setCookie(c, a.Settings().ConsoleURL(), LandlordAccessTokenCookie, resp.AccessToken)
		setCookie(c, a.Settings().ConsoleURL(), LandlordIDTokenCookie, resp.IdToken)
		setCookie(c, a.Settings().ConsoleURL(), LandlordRefreshTokenCookie, resp.RefreshToken)
		redirectToAppRoot(c)
	case rest.LoginStatusNewPasswordRequired:
		setCookie(c, a.Settings().ConsoleURL(), LandlordUsernameCookie, req.Username)
		setCookie(c, a.Settings().ConsoleURL(), LandlordChallengeSessionCookie, resp.Session)
		redirectTo(c, "new-password")
	case rest.LoginStatusPasswordResetRequired:
		setCookie(c, a.Settings().ConsoleURL(), LandlordUsernameCookie, req.Username)
		setCookie(c, a.Settings().ConsoleURL(), LandlordChallengeSessionCookie, resp.Session)
		redirectTo(c, "reset-password")
	default:
		zap.L().Error("unexpected login status value",
			zap.String("status", string(resp.Status)))
		c.String(http.StatusInternalServerError, "internal error")
	}
	return nil
}

func handleNewPassword(a App, c *gin.Context) error {
	username, _ := c.Cookie(LandlordUsernameCookie)
	uv := NewUnauthenticatedView(a.Settings()).WithPotentialUsername(username)
	return executeUnauthenticatedView(c, "newPasswordPage", uv)
}

func handleNewPasswordSubmit(a App, c *gin.Context) error {
	var req rest.SetNewPasswordRequest
	req.Username = c.PostForm("username")
	pwdOne := c.PostForm("password-one")
	pwdTwo := c.PostForm("password-two")
	if pwdOne != pwdTwo {
		c.String(http.StatusBadRequest, "passwords do not match")
		return nil
	}
	req.Password = pwdOne
	req.Session, _ = c.Cookie(LandlordChallengeSessionCookie)
	resp, err := rest.HandleSetNewPassword(a, c, req)
	if err != nil {
		return err
	}
	if resp.Status == rest.LoginStatusMFASetupRequired {
		setCookie(c, a.Settings().ConsoleURL(), LandlordOTPURI, resp.OtpURI)
		setCookie(c, a.Settings().ConsoleURL(), LandlordAccessTokenCookie, resp.AccessToken)
		redirectTo(c, "setup-otp")
		return nil
	}
	if resp.Status != rest.LoginStatusSuccess {
		zap.L().Info("error setting new password",
			zap.String("username", req.Username),
			zap.String("status", string(resp.Status)))
		c.String(http.StatusUnauthorized, "unauthorized")
		return nil
	}
	setCookie(c, a.Settings().ConsoleURL(), LandlordIDTokenCookie, resp.IdToken)
	setCookie(c, a.Settings().ConsoleURL(), LandlordAccessTokenCookie, resp.AccessToken)
	setCookie(c, a.Settings().ConsoleURL(), LandlordRefreshTokenCookie, resp.RefreshToken)
	clearCookie(c, a.Settings().ConsoleURL(), LandlordChallengeSessionCookie)
	redirectToAppRoot(c)
	return nil
}

func handleRecovery(a App, c *gin.Context) error {
	username, _ := c.Cookie(LandlordUsernameCookie)
	uv := NewUnauthenticatedView(a.Settings()).WithPotentialUsername(username)
	return executeUnauthenticatedView(c, "recoveryPage", uv)
}

func handleRecoverySubmit(a App, c *gin.Context) error {
	var req rest.RecoverPasswordRequest
	req.Username = c.PostForm("username")
	resp, err := rest.HandleRecoverPassword(a, c, req)
	if err != nil {
		return err
	}
	// TODO: put the details from the response into front-end
	_ = resp
	setCookie(c, a.Settings().ConsoleURL(), LandlordUsernameCookie, req.Username)
	redirectTo(c, "reset-password")
	return err
}

func handleResetPassword(a App, c *gin.Context) error {
	username, _ := c.Cookie(LandlordUsernameCookie)
	uv := NewUnauthenticatedView(a.Settings()).WithPotentialUsername(username)
	return executeUnauthenticatedView(c, "resetPasswordPage", uv)
}

func handleResetPasswordSubmit(a App, c *gin.Context) error {
	var req rest.ResetPasswordRequest
	req.Username = c.PostForm("username")
	req.ConfirmationCode = c.PostForm("confirmation-code")
	pwdOne := c.PostForm("password-one")
	pwdTwo := c.PostForm("password-two")
	if pwdOne != pwdTwo {
		c.String(http.StatusBadRequest, "passwords do not match")
		return nil
	}
	req.NewPassword = pwdOne
	if err := rest.HandleResetPassword(a, c, req); err != nil {
		return err
	}
	redirectToLogin(c)
	return nil
}
