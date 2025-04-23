package main

import (
	"github.com/apresdev/landlord/internal/appenv"

	"github.com/apresdev/landlord/cmd"
)

var (
	BuildTime  string
	CommitHash string
	GoVersion  string
)

func main() {
	appenv.SetVersionInfo(appenv.VersionInfo{
		Commit:    CommitHash,
		BuildTime: BuildTime,
		GoVersion: GoVersion,
	})
	cmd.Execute()
}
