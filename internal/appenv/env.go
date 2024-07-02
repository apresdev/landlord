package appenv

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

var (
	Debug      bool
	LocalStack bool
)

func loadIfExists(filename string) error {
	if _, err := os.Stat(filename); err == nil {
		return godotenv.Load(filename)
	}
	return nil
}

func init() {
	// See
	// https://github.com/joho/godotenv?tab=readme-ov-file#precedence--conventions
	// for a discussion of .env precedence.
	env := os.Getenv("LANDLORD_ENV")
	if "" == env {
		env = "development"
	}
	err := loadIfExists(".env." + env + ".local")
	if err != nil {
		panic(err)
	}
	if "test" != env {
		err = loadIfExists(".env.local")
		if err != nil {
			panic(err)
		}
	}
	err = loadIfExists(".env." + env)
	if err != nil {
		panic(err)
	}
	err = loadIfExists(".env")
	if err != nil {
		panic(err)
	}
	s := os.Getenv("AWS_PROFILE")
	if s != "" {
		LocalStack = strings.ToLower(s) == "localstack"
	}
	s = os.Getenv("LANDLORD_DEBUG")
	if s != "" {
		Debug = strings.ToLower(s) == "true"
	}
}
