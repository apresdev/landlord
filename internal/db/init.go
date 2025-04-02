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

type Context struct {
	tableName string
	ddb       AWSDynamoDB
}

func NewContext(tableName string, ddb AWSDynamoDB) *Context {
	return &Context{
		tableName: tableName,
		ddb:       ddb,
	}
}

func (c *Context) TableName() string {
	return c.tableName
}

func (c *Context) DDB() AWSDynamoDB {
	return c.ddb
}

func MustInit(ctx context.Context, settings appenv.Settings, ddb AWSDynamoDB) *Context {
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
	return NewContext(settings.DBTableName(), ddb)
}
