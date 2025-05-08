package server

import (
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	ginzap "github.com/gin-contrib/zap"

	"github.com/apresdev/landlord/internal/appenv"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func ipAllowList(al map[string]bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		if !al[clientIP] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"status":  http.StatusForbidden,
				"message": "Permission denied",
			})
			zap.L().Info("Permission denied", zap.String("clientIP", clientIP))
			return
		}
	}
}

func engine() (*gin.Engine, error) {
	gin.SetMode(gin.ReleaseMode)
	if appenv.DevMode() {
		gin.SetMode(gin.DebugMode)
	}
	r := gin.New()

	healthCheckPaths := []string{"/health"}
	tracker := &healthCheckLogTracker{}

	// - Logs all requests, like a combined access and error log
	// - Logs to stdout
	// - RFC3339 with UTC time format
	// - Logs stack traces on panic
	r.Use(ginzap.GinzapWithConfig(zap.L(), &ginzap.Config{
		TimeFormat: time.RFC3339,
		UTC:        true,
		Skipper: func(c *gin.Context) bool {
			path := c.Request.URL.Path
			for _, healthPath := range healthCheckPaths {
				if path == healthPath {
					tracker.logHealthCheck()
					return true // Skip logging
				}
			}
			// Special case for AWS health checks aimed at the root
			if path == "/" {
				ua := c.GetHeader("User-Agent")
				if strings.Contains(strings.ToLower(ua), "healthcheck") {
					tracker.logHealthCheck()
					return true // Skip logging
				}
			}
			return false
		},
	}))
	r.Use(ginzap.RecoveryWithZap(zap.L(), true))

	ips := os.Getenv("LANDLORD_IP_ALLOW_LIST")
	if ips != "" {
		al := make(map[string]bool)
		al["127.0.0.1"] = true
		for _, ip := range strings.Split(ips, ",") {
			ip = strings.TrimSpace(ip)
			al[ip] = true
		}
		r.Use(ipAllowList(al))
	}
	if err := r.SetTrustedProxies(nil); err != nil {
		return nil, err
	}
	return r, nil
}

type ListenAddr struct {
	UseTLS bool
	Addr   string
}

func parseListenAddr(s string) (ListenAddr, error) {
	if strings.HasPrefix(s, "https://") {
		return ListenAddr{UseTLS: true, Addr: s[8:]}, nil
	}
	if strings.HasPrefix(s, "http://") {
		return ListenAddr{UseTLS: false, Addr: s[7:]}, nil
	}
	return ListenAddr{UseTLS: false, Addr: s}, nil
}

type healthCheckLogTracker struct {
	mu                 sync.Mutex
	healthCheckCount   int
	lastHealthCheckLog time.Time
}

const (
	healthCheckLogFrequency = 10
	healthCheckLogTimeFlush = 1 * time.Minute
)

// logHealthCheck logs the health check request if it meets certain conditions.
func (h *healthCheckLogTracker) logHealthCheck() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.healthCheckCount++
	now := time.Now()
	lastLog := h.lastHealthCheckLog
	shouldLog := h.healthCheckCount%healthCheckLogFrequency == 0 || now.Sub(lastLog) >= healthCheckLogTimeFlush
	if shouldLog {
		h.lastHealthCheckLog = now
		zap.L().Info("Health check accumulator",
			zap.Int("count", h.healthCheckCount))
	}
}
