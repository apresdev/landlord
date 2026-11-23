package apptest

import (
	"context"

	"github.com/apresdev/landlord/internal/appenv"
	"github.com/apresdev/landlord/internal/apperr"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"go.uber.org/zap"
)

// InitEmulator initializes the AWS emulator environment for testing. It deletes
// and recreates the DynamoDB table and the Cognito user pool. This is meant for
// testing only and only works against the AWS Emulator (see scripts/aws-emulator),
// selected with AWS_ENDPOINT_URL. It returns true if the initialization was
// successful, false otherwise. False indicates that the emulator is not running
// and the test should be skipped.
func InitEmulator(settings appenv.Settings) bool {
	cfg, err := emulatorConfig()
	if err != nil {
		return false
	}
	dangerousRecreateDynamoTable(cfg, settings.DBTableName())
	dangerousRecreateUserPool(cfg, settings.UserPoolName())
	return true
}

// ResetEmulator resets the emulator environment for testing. It deletes all
// items in the DynamoDB table and recreates the Cognito user pool.
func ResetEmulator(settings appenv.Settings) {
	cfg := mustEmulatorConfig()
	dangerousDeleteAllItems(cfg, settings.DBTableName())
	dangerousRecreateUserPool(cfg, settings.UserPoolName())
}

func emulatorConfig() (aws.Config, error) {
	if !emulatorIsRunning() {
		return aws.Config{}, apperr.NewInternal("AWS emulator not running")
	}
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		zap.L().Fatal("unable to load SDK config", zap.Error(err))
	}
	return cfg, nil
}

func mustEmulatorConfig() aws.Config {
	cfg, err := emulatorConfig()
	if err != nil {
		zap.L().Fatal("failed to load emulator config", zap.Error(err))
	}
	return cfg
}
