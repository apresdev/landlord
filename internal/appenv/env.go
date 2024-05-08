package appenv

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

var Debug bool
var Localstack bool

func init() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}
	s := os.Getenv("AWS_PROFILE")
	if s != "" {
		Localstack = strings.ToLower(s) == "localstack"
	}
	s = os.Getenv("APRES_DEBUG")
	if s != "" {
		Debug = strings.ToLower(s) == "true"
	}
}
