package webapp

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/apresdev/landlord/internal/apperr"

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

	username, err := getEncodedCookie(c, LandlordUsernameCookie)
	if err != nil || username == "" {
		redirectToLogin(c)
		return nil
	}

	accessToken, err := getEncodedCookie(c, LandlordAccessTokenCookie)
	if err != nil || accessToken == "" {
		clearLandlordCookies(c, a.Settings().ConsoleURL())
		redirectToLogin(c)
		return nil
	}

	_, tenant, err := db.GetUserAndTenant(c.Request.Context(), a.DBCtx(), username)
	if err != nil {
		zap.L().Warn("error finding tenant for username",
			zap.String("username", username),
			zap.Error(err))
		clearLandlordCookies(c, a.Settings().ConsoleURL())
		redirectToLogin(c)
		return nil
	}
	validationURL := jwx.ValidationURL(a.Cfg().Region, tenant.UserPoolID)
	_, err = a.Validator().ValidateToken(
		c.Request.Context(), clientID(c), validationURL, accessToken)
	if err != nil {
		redirectToLogin(c)
		return err
	}

	redirectTo(c, "admin/dashboard")

	return nil
}

func handleLogin(a App, c *gin.Context) error {
	username, _ := getEncodedCookie(c, LandlordUsernameCookie)
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
		setEncodedCookie(a, c, LandlordUsernameCookie, req.Username)
		setEncodedCookie(a, c, LandlordAccessTokenCookie, resp.AccessToken)
		setEncodedCookie(a, c, LandlordRefreshTokenCookie, resp.RefreshToken)
		redirectToAppRoot(c)
		return nil
	case rest.LoginStatusNewPasswordRequired:
		setEncodedCookie(a, c, LandlordUsernameCookie, req.Username)
		setEncodedCookie(a, c, LandlordChallengeSessionCookie, resp.Session)
		redirectTo(c, "new-password")
		return nil
	case rest.LoginStatusPasswordResetRequired:
		setEncodedCookie(a, c, LandlordUsernameCookie, req.Username)
		setEncodedCookie(a, c, LandlordChallengeSessionCookie, resp.Session)
		redirectTo(c, "reset-password")
		return nil
	case rest.LoginStatusMFASetupRequired:
		setEncodedCookie(a, c, LandlordUsernameCookie, req.Username)
		setEncodedCookie(a, c, LandlordAccessTokenCookie, resp.AccessToken)
		setEncodedCookie(a, c, LandlordOTPURICookie, resp.OtpURI)
		redirectTo(c, "setup-otp")
		return nil
	case rest.LoginStatusMFACodeRequired:
		setEncodedCookie(a, c, LandlordUsernameCookie, req.Username)
		setEncodedCookie(a, c, LandlordChallengeSessionCookie, resp.Session)
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
		return apperr.NewBadRequest("missing uri")
	}
	b, err := base64.URLEncoding.DecodeString(uri)
	if err != nil {
		return apperr.NewBadRequest("invalid uri")
	}
	png, err := qrcode.Encode(string(b), qrcode.Medium, 256)
	if err != nil {
		return err
	}
	c.Data(http.StatusOK, "image/png", png)
	return nil
}

func handleSetupOTP(a App, c *gin.Context) error {
	otpURI, err := getEncodedCookie(c, LandlordOTPURICookie)
	if err != nil || otpURI == "" {
		zap.L().Error("missing OTP URI in handleSetupOTP",
			zap.String("cid", string(clientID(c))),
			zap.Error(err),
			zap.String("cookie", string(LandlordOTPURICookie)))
		return apperr.NewUnauthorized("unauthorized")
	}
	otpImageURI := appPath(fmt.Sprintf(
		"/qrcode?uri=%s", base64.URLEncoding.EncodeToString([]byte(otpURI))))
	username, err := getEncodedCookie(c, LandlordUsernameCookie)
	if err != nil || username == "" {
		zap.L().Error("missing username in handleSetupOTP",
			zap.String("cid", string(clientID(c))),
			zap.Error(err),
			zap.String("cookie", string(LandlordUsernameCookie)))
		return apperr.NewUnauthorized("unauthorized")
	}
	uv := NewUnauthenticatedView(a.Settings()).
		WithPotentialUsername(username).WithOTPImageLink(otpImageURI)
	return executeUnauthenticatedView(c, "setupOTPPage", uv)
}

func handleSetupOTPSubmit(a App, c *gin.Context) error {
	var req rest.VerifyOtpAssociationRequest
	req.Username = c.PostForm("username")
	req.OtpCode = c.PostForm("code")
	var err error
	req.AccessToken, err = getEncodedCookie(c, LandlordAccessTokenCookie)
	if err != nil || req.AccessToken == "" {
		return apperr.NewBadRequest("missing access token")
	}
	err = rest.HandleVerifyOTPAssociation(a, c, req)
	if err != nil {
		zap.L().Error("error verifying OTP association",
			zap.String("username", req.Username),
			zap.Error(err))
		return apperr.NewBadRequest("error verifying OTP association")
	}
	// TODO: add refresh token to request and re-fetch tokens here
	redirectToAppRoot(c)
	return nil
}

func handleOTP(a App, c *gin.Context) error {
	username, err := getEncodedCookie(c, LandlordUsernameCookie)
	if err != nil || username == "" {
		return apperr.NewBadRequest("missing username")
	}
	uv := NewUnauthenticatedView(a.Settings()).WithPotentialUsername(username)
	return executeUnauthenticatedView(c, "otpCodePage", uv)
}

func handleOTPSubmit(a App, c *gin.Context) error {
	var req rest.ValidateMfaCodeRequest
	req.Username = c.PostForm("username")
	req.MfaCode = c.PostForm("code")
	var err error
	req.Session, err = getEncodedCookie(c, LandlordChallengeSessionCookie)
	if err != nil || req.Session == "" {
		return apperr.NewBadRequest("missing session")
	}
	resp, err := rest.HandleValidateMFACode(a, c, req)
	if err != nil {
		return err
	}
	switch resp.Status {
	case rest.LoginStatusSuccess:
		setEncodedCookie(a, c, LandlordUsernameCookie, req.Username)
		setEncodedCookie(a, c, LandlordAccessTokenCookie, resp.AccessToken)
		setEncodedCookie(a, c, LandlordRefreshTokenCookie, resp.RefreshToken)
		clearCookie(c, a.Settings().ConsoleURL(), LandlordChallengeSessionCookie)
		redirectToAppRoot(c)
	case rest.LoginStatusNewPasswordRequired:
		setEncodedCookie(a, c, LandlordUsernameCookie, req.Username)
		setEncodedCookie(a, c, LandlordChallengeSessionCookie, resp.Session)
		redirectTo(c, "new-password")
	case rest.LoginStatusPasswordResetRequired:
		setEncodedCookie(a, c, LandlordUsernameCookie, req.Username)
		setEncodedCookie(a, c, LandlordChallengeSessionCookie, resp.Session)
		redirectTo(c, "reset-password")
	default:
		zap.L().Error("unexpected login status value",
			zap.String("status", string(resp.Status)))
		return apperr.NewInternal("unexpected login status value")
	}
	return nil
}

func handleNewPassword(a App, c *gin.Context) error {
	username, err := getEncodedCookie(c, LandlordUsernameCookie)
	if err != nil || username == "" {
		return apperr.NewBadRequest("missing username")
	}
	uv := NewUnauthenticatedView(a.Settings()).WithPotentialUsername(username)
	return executeUnauthenticatedView(c, "newPasswordPage", uv)
}

func handleNewPasswordSubmit(a App, c *gin.Context) error {
	var req rest.SetNewPasswordRequest
	req.Username = c.PostForm("username")
	pwdOne := c.PostForm("password-one")
	pwdTwo := c.PostForm("password-two")
	if pwdOne != pwdTwo {
		return apperr.NewBadRequest("passwords do not match")
	}
	req.Password = pwdOne
	var err error
	req.Session, err = getEncodedCookie(c, LandlordChallengeSessionCookie)
	if err != nil || req.Session == "" {
		return apperr.NewBadRequest("missing session")
	}
	resp, err := rest.HandleSetNewPassword(a, c, req)
	if err != nil {
		return err
	}
	if resp.Status == rest.LoginStatusMFASetupRequired {
		setEncodedCookie(a, c, LandlordOTPURICookie, resp.OtpURI)
		setEncodedCookie(a, c, LandlordAccessTokenCookie, resp.AccessToken)
		redirectTo(c, "setup-otp")
		return nil
	}
	if resp.Status != rest.LoginStatusSuccess {
		zap.L().Info("error setting new password",
			zap.String("username", req.Username),
			zap.String("status", string(resp.Status)))
		return apperr.NewBadRequest("unauthorized")
	}
	setEncodedCookie(a, c, LandlordAccessTokenCookie, resp.AccessToken)
	setEncodedCookie(a, c, LandlordRefreshTokenCookie, resp.RefreshToken)
	clearCookie(c, a.Settings().ConsoleURL(), LandlordChallengeSessionCookie)
	redirectToAppRoot(c)
	return nil
}

func handleRecovery(a App, c *gin.Context) error {
	username, err := getEncodedCookie(c, LandlordUsernameCookie)
	if err != nil || username == "" {
		return apperr.NewBadRequest("missing username")
	}
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
	setEncodedCookie(a, c, LandlordUsernameCookie, req.Username)
	redirectTo(c, "reset-password")
	return err
}

func handleResetPassword(a App, c *gin.Context) error {
	username, err := getEncodedCookie(c, LandlordUsernameCookie)
	if err != nil || username == "" {
		return apperr.NewBadRequest("missing username")
	}
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
		return apperr.NewBadRequest("passwords do not match")
	}
	req.NewPassword = pwdOne
	if err := rest.HandleResetPassword(a, c, req); err != nil {
		return err
	}
	redirectToLogin(c)
	return nil
}
