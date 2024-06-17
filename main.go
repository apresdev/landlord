package main

import (
	"github.com/apresdev/landlord/cmd"
	"github.com/apresdev/landlord/internal/appenv"
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
	cmd.Execute()
}
