package server

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/a-h/templ"

	"go.uber.org/zap"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/gin-gonic/gin"
)

func Run() error {
	r, err := engine()
	if err != nil {
		return err
	}
	cs := MustClients()
	r.GET("/", func(c *gin.Context) {
		handleRoot(c, cs.Cfg(), cs.CogIdp(), cs.Validator())
	})
	r.GET("/login", func(c *gin.Context) {
		render(c, loginPage())
	})
	r.POST("/login/submit", func(c *gin.Context) {
		fmt.Println(c.PostForm("username"))
		fmt.Println(c.PostForm("password"))
		c.Redirect(http.StatusFound, "/")
	})

	r.GET("/username", func(c *gin.Context) {
		render(c, usernamePage())
	})
	r.POST("/username/submit", func(c *gin.Context) {
		username := c.PostForm("username")
		tens, err := authx.LoadTenants(
			context.Background(), cs.Cfg(), cs.Sts(), cs.CogIdp())
		if err != nil {
			c.String(http.StatusInternalServerError,
				"Error loading tenants: %v", err)
			return
		}
		ts, err := tens.ForUsername(username)
		if err != nil {
			c.String(http.StatusInternalServerError,
				"Error finding tenant for username: %v", err)
			return
		}
		if ts == nil || len(ts) == 0 {
			zap.L().Info("No tenants found for username", zap.String("username", username))
			setUsernameCookie(c, username)
			c.Redirect(http.StatusFound, "/signup")
			return
		} else {
			// TODO: redirect to login if the user has not confirmed, etc.
			render(c, alreadySignedUpPage(username))
		}
	})
	r.GET("/signup", func(c *gin.Context) {
		username, _ := c.Cookie(LandlordUsernameCookie)
		if username != "" {
			tens, err := authx.LoadTenants(
				context.Background(), cs.Cfg(), cs.Sts(), cs.CogIdp())
			if err != nil {
				c.String(http.StatusInternalServerError,
					"Error loading tenants: %v", err)
				return
			}
			ten, err := tens.ForExclusiveEmail(username)
			if err != nil {
				render(c, adminHelpPage(username, "admin@apres.dev")) //TODO
			} else {
				if ten.HasUser(username) {
					render(c, alreadySignedUpPage(username))
				} else {
					render(c, signupPage(username))
				}
			}
		} else {
			render(c, signupPage(username))
		}
	})
	r.POST("/signup/submit", func(c *gin.Context) {
		username := c.PostForm("username")
		tens, err := authx.LoadTenants(
			context.Background(), cs.Cfg(), cs.Sts(), cs.CogIdp())
		if err != nil {
			c.String(http.StatusInternalServerError,
				"Error loading tenants: %v", err)
			return
		}
		err = authx.SendInvite(
			c.Request.Context(), cs.SES(), cs.CogIdp(), tens, username, authx.RoleGuest)
		if err != nil {
			// TODO: handle this kind of error in the admin panel
			c.String(http.StatusInternalServerError,
				"Error routing invite: %v", err)
			return
		}
		zap.L().Info("Signup form submitted", zap.String("username", username))
		setUsernameCookie(c, username)
		c.Redirect(http.StatusFound, "/signup/confirm")
	})
	r.GET("/signup/confirm", func(c *gin.Context) {
		username, _ := c.Cookie(LandlordUsernameCookie)
		if err := signupConfirmPage(username).Render(c.Request.Context(), c.Writer); err != nil {
			c.String(http.StatusInternalServerError,
				"Error rendering signup confirm form: %v", err)
		}
	})
	r.POST("/signup/resend", func(c *gin.Context) {
		username := c.PostForm("username")
		tens, err := authx.LoadTenants(
			context.Background(), cs.Cfg(), cs.Sts(), cs.CogIdp())
		if err != nil {
			c.String(http.StatusInternalServerError,
				"Error loading tenants: %v", err)
			return
		}
		err = authx.ResendInvite(
			c.Request.Context(), cs.SES(), cs.CogIdp(), tens, username, authx.RoleGuest)
		if err != nil {
			c.String(http.StatusInternalServerError,
				"Error routing invite: %v", err)
			return
		}
		c.Redirect(http.StatusFound, "/signup/confirm")
	})
	r.GET("/logout", func(c *gin.Context) {
		clearCookies(c,
			LandlordUsernameCookie,
			LandlordIDTokenCookie,
			LandlordAccessTokenCookie,
			LandlordRefreshTokenCookie)
		c.Redirect(http.StatusFound, "/")
	})

	listenAddr, err := parseListenAddr(os.Getenv("LANDLORD_LISTEN_ADDR"))
	if err != nil {
		return err
	}
	if listenAddr.UseTLS {
		return runTLS(listenAddr.Addr, r)
	} else {
		return runLocal(listenAddr.Addr, r)
	}
}

func runTLS(addr string, r *gin.Engine) error {
	certPath := os.Getenv("LANDLORD_CERT_PATH")
	if certPath == "" {
		return fmt.Errorf("LANDLORD_CERT_PATH must be set")
	}
	keyPath := os.Getenv("LANDLORD_PRIVATE_KEY_PATH")
	if keyPath == "" {
		return fmt.Errorf("LANDLORD_PRIVATE_KEY_PATH must be set")
	}
	if err := r.RunTLS(addr, certPath, keyPath); err != nil {
		return err
	}
	return nil
}

func runLocal(addr string, r *gin.Engine) error {
	if err := r.Run(addr); err != nil {
		return err
	}
	return nil
}

func setUsernameCookie(c *gin.Context, username string) {
	c.SetCookie(
		LandlordUsernameCookie,
		username,
		LandlordCookieMaxAge,
		"/",
		authx.AppDomainHost(),
		true,
		true)
}

func clearCookies(c *gin.Context, cookies ...string) {
	for _, cookie := range cookies {
		c.SetCookie(
			cookie,
			"",
			-1,
			"/",
			authx.AppDomainHost(),
			true,
			true)
	}
}

func render(c *gin.Context, t templ.Component) {
	if err := t.Render(c.Request.Context(), c.Writer); err != nil {
		c.String(http.StatusInternalServerError,
			"Error rendering page: %v", err)
		zap.L().Error("Error rendering page", zap.Error(err))
	}
}
