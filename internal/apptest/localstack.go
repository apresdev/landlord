package apptest

import (
	"context"
	"os"

	"github.com/apresdev/landlord/internal/appenv"
	"github.com/apresdev/landlord/internal/apperr"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"go.uber.org/zap"
)

// InitLocalStack initializes the localstack environment for testing.
// It deletes and recreates the DynamoDB table and the Cognito user pool.
// This is meant for testing only and only works in a localstack environment.
// It returns true if the initialization was successful, false otherwise.
// False indicates that localstack is not running and the test should be
// skipped.
func InitLocalStack(settings appenv.Settings) bool {
	cfg, err := localStackConfig()
	if err != nil {
		return false
	}
	dangerousRecreateDynamoTable(cfg, settings.DBTableName())
	dangerousRecreateUserPool(cfg, settings.UserPoolName())
	return true
}

// ResetLocalStack resets the localstack environment for testing.
// It deletes all items in the DynamoDB table and recreates the Cognito user pool.
func ResetLocalStack(settings appenv.Settings) {
	cfg := mustLocalStackConfig()
	dangerousDeleteAllItems(cfg, settings.DBTableName())
	dangerousRecreateUserPool(cfg, settings.UserPoolName())
}

func localStackConfig() (aws.Config, error) {
	if !localstackIsRunning() {
		return aws.Config{}, apperr.NewInternal("localstack not running")
	}
	if os.Getenv("AWS_PROFILE") != "localstack" {
		zap.L().Fatal("AWS_PROFILE must be set to 'localstack'")
	}
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		zap.L().Fatal("unable to load SDK config", zap.Error(err))
	}
	return cfg, nil
}

func mustLocalStackConfig() aws.Config {
	cfg, err := localStackConfig()
	if err != nil {
		zap.L().Fatal("failed to load localstack config", zap.Error(err))
	}
	return cfg
}
