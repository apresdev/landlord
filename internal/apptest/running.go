package apptest

import (
	"errors"
	"net/http"
	"net/url"
	"os"
	"syscall"

	"go.uber.org/zap"
)

const DefaultLocalstackEndpoint = "http://localhost:4566"

func Endpoint() string {
	v := os.Getenv("LANDLORD_LOCALSTACK_ENDPOINT")
	if v != "" {
		return v
	}
	return DefaultLocalstackEndpoint
}

var isRunningDetermined bool
var isRunning bool

// localstackIsRunning checks the localstack health check endpoint to see if it is running
func localstackIsRunning() bool {
	if isRunningDetermined {
		return isRunning
	}
	u, err := url.JoinPath(Endpoint(), "_localstack/health")
	if err != nil {
		zap.L().Error("Error checking LocalStack:", zap.Error(err))
		return false
	}
	response, err := http.Get(u)
	defer func() {
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
	}()
	if err != nil {
		var syscallErr syscall.Errno
		if errors.As(err, &syscallErr) && errors.Is(syscallErr, syscall.ECONNREFUSED) {
			zap.L().Info("LocalStack is not running", zap.Error(err))
			isRunning = false
			isRunningDetermined = true
			return false
		}
		zap.L().Error("Error checking LocalStack:", zap.Error(err))
	}
	isRunning = response.StatusCode == http.StatusOK
	isRunningDetermined = true
	return isRunning
}
