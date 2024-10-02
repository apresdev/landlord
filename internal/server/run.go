package server

import (
	"fmt"
	"os"

	"github.com/apresdev/landlord/internal/appcore"

	"go.uber.org/zap"

	"github.com/apresdev/landlord/internal/webapp"

	"github.com/apresdev/landlord/internal/appenv"
	"github.com/apresdev/landlord/internal/rest"

	"github.com/gin-gonic/gin"
)

func Run() error {
	baseRoutes, err := engine()
	if err != nil {
		return err
	}

	a := appcore.Must()

	if a.Settings().Routes() == appenv.RouteSettingBoth {
		rest.AddRoutes(a, baseRoutes)
		webapp.AddRoutes(a, baseRoutes)
	} else if a.Settings().Routes() == appenv.RouteSettingRest {
		rest.AddRoutes(a, baseRoutes)
	} else if a.Settings().Routes() == appenv.RouteSettingWebApp {
		webapp.AddRoutes(a, baseRoutes)
	} else {
		zap.L().Fatal("Invalid route setting", zap.Any("setting", a.Settings().Routes()))
	}

	listenAddr, err := parseListenAddr(os.Getenv("LANDLORD_LISTEN_ADDR"))
	if err != nil {
		return err
	}
	if listenAddr.UseTLS {
		return runTLS(listenAddr.Addr, baseRoutes)
	} else {
		return runLocal(listenAddr.Addr, baseRoutes)
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
