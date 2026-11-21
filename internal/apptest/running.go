package apptest

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/url"
	"os"
	"time"

	"go.uber.org/zap"
)

const DefaultLocalStackEndpoint = "http://localhost:4566"

func init() {
	trustCABundle()
}

// trustCABundle makes the default HTTP transport (used, for example, to fetch
// the JWKS) trust the CA in AWS_CA_BUNDLE. The AWS SDK honors that variable
// itself, but net/http does not, and a local AWS emulator serves a self-signed
// certificate.
func trustCABundle() {
	path := os.Getenv("AWS_CA_BUNDLE")
	if path == "" {
		return
	}
	pem, err := os.ReadFile(path)
	if err != nil {
		zap.L().Warn("could not read AWS_CA_BUNDLE", zap.Error(err))
		return
	}
	pool, err := x509.SystemCertPool()
	if err != nil || pool == nil {
		pool = x509.NewCertPool()
	}
	pool.AppendCertsFromPEM(pem)
	if t, ok := http.DefaultTransport.(*http.Transport); ok {
		t = t.Clone()
		t.TLSClientConfig = &tls.Config{RootCAs: pool}
		http.DefaultTransport = t
	}
}

// Endpoint returns the endpoint of the local AWS stand-in (LocalStack or the
// AWS Emulator): LANDLORD_LOCALSTACK_ENDPOINT, else AWS_ENDPOINT_URL, else the
// LocalStack default.
func Endpoint() string {
	if v := os.Getenv("LANDLORD_LOCALSTACK_ENDPOINT"); v != "" {
		return v
	}
	if v := os.Getenv("AWS_ENDPOINT_URL"); v != "" {
		return v
	}
	return DefaultLocalStackEndpoint
}

var isRunningDetermined bool
var isRunning bool

// healthPaths are the health check paths of LocalStack and the AWS Emulator.
var healthPaths = []string{"_localstack/health", "__admin/health"}

// localstackIsRunning checks the health check endpoints to see if the local
// AWS stand-in is running. Unreachable or unhealthy means "not running" so that
// dependent tests skip.
func localstackIsRunning() bool {
	if isRunningDetermined {
		return isRunning
	}
	isRunningDetermined = true
	client := &http.Client{Timeout: 5 * time.Second}
	for _, p := range healthPaths {
		u, err := url.JoinPath(Endpoint(), p)
		if err != nil {
			zap.L().Error("Error checking local AWS endpoint:", zap.Error(err))
			return false
		}
		response, err := client.Get(u)
		if err != nil {
			continue
		}
		_ = response.Body.Close()
		if response.StatusCode == http.StatusOK {
			isRunning = true
			return true
		}
	}
	zap.L().Info("local AWS endpoint is not running", zap.String("endpoint", Endpoint()))
	return false
}
