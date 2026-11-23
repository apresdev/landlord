package apptest

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/apresdev/landlord/internal/appenv"
	"go.uber.org/zap"
)

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

// Endpoint returns the endpoint of the local AWS emulator, from the standard
// AWS_ENDPOINT_URL variable.
func Endpoint() string {
	return os.Getenv("AWS_ENDPOINT_URL")
}

var isRunningDetermined bool
var isRunning bool

// emulatorIsRunning checks the AWS Emulator health endpoint. A missing endpoint,
// or an unreachable or unhealthy emulator, means "not running" so that dependent
// tests skip.
func emulatorIsRunning() bool {
	if isRunningDetermined {
		return isRunning
	}
	isRunningDetermined = true
	// Explicit opt-in: these tests delete and recreate tables and user pools, so
	// never run them against an endpoint that is not known to be an emulator.
	if !appenv.LocalAWS() || Endpoint() == "" {
		zap.L().Info("LANDLORD_AWS_EMULATOR and AWS_ENDPOINT_URL are not set; skipping AWS emulator tests")
		return false
	}
	u, err := url.JoinPath(Endpoint(), "__admin/health")
	if err != nil {
		zap.L().Error("Error checking AWS emulator:", zap.Error(err))
		return false
	}
	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Get(u)
	if err != nil {
		zap.L().Info("AWS emulator is not running", zap.String("endpoint", Endpoint()), zap.Error(err))
		return false
	}
	_ = response.Body.Close()
	isRunning = response.StatusCode == http.StatusOK
	return isRunning
}
