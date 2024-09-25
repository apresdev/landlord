package db

import (
	"context"
	"errors"

	"github.com/apresdev/landlord/internal/appenv"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"go.uber.org/zap"
)

var tableName string

func MustInit(ctx context.Context, settings appenv.Settings, ddb AWSDynamoDB) {
	_, err := ddb.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(settings.DBTableName()),
	})
	var notFound *types.ResourceNotFoundException
	if err != nil {
		if errors.As(err, &notFound) {
			zap.L().Fatal("landlord's DynamoDB table not found",
				zap.String("table", settings.DBTableName()),
				zap.Error(err))
		}
		zap.L().Fatal("error describing landlord's DynamoDB table",
			zap.String("table", settings.DBTableName()),
			zap.Error(err))
	}
	tableName = settings.DBTableName()
}

func TableName() string {
	return tableName
}

func resolveStr(v types.AttributeValue) (string, error) {
	if v == nil {
		return "", errors.New("unexpected nil attribute value")
	}
	if v1, ok := v.(*types.AttributeValueMemberS); !ok {
		return "", errors.New("expected string attribute value")
	} else {
		return v1.Value, nil
	}
}
