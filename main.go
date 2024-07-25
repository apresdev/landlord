package main

import (
	"github.com/apresdev/landlord/internal/appenv"

	"github.com/apresdev/landlord/cmd"
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
