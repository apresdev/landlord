package db

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

//go:generate mockery --output ../mocks --name AWSDynamoDB --with-expecter=true
type AWSDynamoDB interface {
	CreateTable(
		ctx context.Context,
		params *dynamodb.CreateTableInput,
		optFns ...func(*dynamodb.Options)) (
		*dynamodb.CreateTableOutput, error)
	DescribeTable(
		ctx context.Context,
		params *dynamodb.DescribeTableInput,
		optFns ...func(*dynamodb.Options)) (
		*dynamodb.DescribeTableOutput, error)
	DeleteTable(
		ctx context.Context,
		params *dynamodb.DeleteTableInput,
		optFns ...func(*dynamodb.Options)) (
		*dynamodb.DeleteTableOutput, error)
	PutItem(
		ctx context.Context,
		params *dynamodb.PutItemInput,
		optFns ...func(*dynamodb.Options)) (
		*dynamodb.PutItemOutput, error)
	GetItem(
		ctx context.Context,
		params *dynamodb.GetItemInput,
		optFns ...func(*dynamodb.Options)) (
		*dynamodb.GetItemOutput, error)
	DeleteItem(
		ctx context.Context,
		params *dynamodb.DeleteItemInput,
		optFns ...func(*dynamodb.Options)) (
		*dynamodb.DeleteItemOutput, error)
	Query(
		ctx context.Context,
		params *dynamodb.QueryInput,
		optFns ...func(*dynamodb.Options)) (
		*dynamodb.QueryOutput, error)
	Scan(
		ctx context.Context,
		params *dynamodb.ScanInput,
		optFns ...func(*dynamodb.Options)) (
		*dynamodb.ScanOutput, error)
}
