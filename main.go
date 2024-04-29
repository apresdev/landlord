package main

import (
	"github.com/apresdev/landlord/cmd"
	"github.com/apresdev/landlord/internal/appenv"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

var (
	BuildTime  string
	CommitHash string
	GoVersion  string
	GitTag     string
)

func main() {
	appenv.SetVersionInfo(appenv.VersionInfo{
		Version:   GitTag,
		Commit:    CommitHash,
		BuildTime: BuildTime,
		GoVersion: GoVersion,
	})
	appenv.MustConfigureLogging()
	if err := godotenv.Load(); err != nil {
		zap.L().Error("failed to load env vars", zap.Error(err))
	}
	cmd.Execute()
}
