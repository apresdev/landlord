package localstack

import (
	"fmt"
	"net/http"
)

// IsRunning checks the localstack health check endpoint to see if it is running
func IsRunning() bool {
	url := "http://localhost:4566/_localstack/health"
	response, err := http.Get(url)
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
