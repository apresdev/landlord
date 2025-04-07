package apptest

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"go.uber.org/zap"

	"github.com/aws/aws-sdk-go-v2/aws"
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

func dangerousRecreateDynamoTable(cfg aws.Config, name string) {
	ddb := dynamodb.NewFromConfig(cfg)
	ctx := context.Background()
	_, err := ddb.DeleteTable(ctx, &dynamodb.DeleteTableInput{
		TableName: aws.String(name),
	})
	if err != nil {
		var notFound *types.ResourceNotFoundException
		if errors.As(err, &notFound) {
			zap.L().Info("table does not exist", zap.String("table", name))
		} else {
			zap.L().Fatal("failed to delete table", zap.Error(err))
		}
	}
	input := &dynamodb.CreateTableInput{
		TableName: aws.String(name),
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("pk"),
				KeyType:       types.KeyTypeHash,
			},
			{
				AttributeName: aws.String("sk"),
				KeyType:       types.KeyTypeRange,
			},
		},
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("pk"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("sk"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("altName"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		BillingMode: types.BillingModePayPerRequest,
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
			{
				IndexName: aws.String("altNameIndex"),
				KeySchema: []types.KeySchemaElement{
					{
						AttributeName: aws.String("altName"),
						KeyType:       types.KeyTypeHash,
					},
				},
				Projection: &types.Projection{
					ProjectionType: types.ProjectionTypeAll,
				},
			},
		},
	}
	if _, err := ddb.CreateTable(ctx, input); err != nil {
		zap.L().Fatal("failed to create table", zap.Error(err))
	}
}

// dangerousDeleteAllItems deletes all items from the DynamoDB table -- only
// for testing.
func dangerousDeleteAllItems(cfg aws.Config, tableName string) {
	ddb := dynamodb.NewFromConfig(cfg)
	scanInput := &dynamodb.ScanInput{
		TableName: aws.String(tableName),
	}
	ctx := context.Background()
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
