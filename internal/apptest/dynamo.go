package apptest

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"go.uber.org/zap"

	"github.com/apresdev/landlord/internal/localstack"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func MustStructToDynamoMap(input any) map[string]types.AttributeValue {
	output, err := attributevalue.MarshalMap(input)
	if err != nil {
		panic(err)
	}
	return output
}

// DangerousDeleteAllItems deletes all items from the DynamoDB table -- only
// for testing
func DangerousDeleteAllItems(tableName string) {
	if os.Getenv("AWS_PROFILE") != "localstack" {
		zap.L().Fatal("AWS_PROFILE must be set to 'localstack'")
	}
	if !localstack.IsRunning() {
		zap.L().Fatal("localstack not running")
	}
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		zap.L().Fatal("unable to load SDK config", zap.Error(err))
	}
	ddb := dynamodb.NewFromConfig(cfg)
	scanInput := &dynamodb.ScanInput{
		TableName: aws.String(tableName),
	}
	scanResult, err := ddb.Scan(ctx, scanInput)
	if err != nil {
		zap.L().Fatal("failed to scan table", zap.Error(err))
	}
	for _, item := range scanResult.Items {
		key := map[string]types.AttributeValue{}
		key["pk"] = item["pk"]
		key["sk"] = item["sk"]
		_, err := ddb.DeleteItem(ctx, &dynamodb.DeleteItemInput{
			TableName: aws.String(tableName),
			Key:       key,
		})
		if err != nil {
			zap.L().Fatal("failed to delete item", zap.Error(err))
		}
	}
}
