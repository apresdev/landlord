package localstack

import (
	"fmt"
	"net/http"
	"net/url"
	"os"

	"go.uber.org/zap"
)

const DefaultLocalstackEndpoint = "http://localhost:4566"

func Endpoint() string {
	v := os.Getenv("APRES_LOCALSTACK_ENDPOINT")
	if v != "" {
		return v
	}
	return DefaultLocalstackEndpoint
}

// IsRunning checks the localstack health check endpoint to see if it is running
func IsRunning() bool {
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
		fmt.Println("Error checking LocalStack:", err)
		return false
	}
	return response.StatusCode == http.StatusOK
}
