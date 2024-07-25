package db

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/apresdev/landlord/internal/apperr"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var tableNames = []string{
	usersTable,
	tenantsTable,
	emailDomainsTable,
}

func TableNames() []string {
	return tableNames
}

func Init(ctx context.Context, ddb AWSDynamoDB) error {
	if err := initUsers(ctx, ddb); err != nil {
		return err
	}
	if err := initTenants(ctx, ddb); err != nil {
		return err
	}
	if err := initEmailDomains(ctx, ddb); err != nil {
		return err
	}
	for {
		allActive := true
		for _, tableName := range tableNames {
			if !tableActive(ctx, ddb, tableName) {
				allActive = false
				fmt.Printf("Waiting for table %s to become active\n", tableName)
			}
		}
		if allActive {
			break
		}
		time.Sleep(2 * time.Second)
	}
	return nil
}

func attrDefEq(a, b []types.AttributeDefinition) bool {
	if len(a) != len(b) {
		return false
	}
	sort.Slice(a, func(i, j int) bool {
		return strings.Compare(
			aws.ToString(a[i].AttributeName),
			aws.ToString(a[j].AttributeName)) > 0
	})
	sort.Slice(b, func(i, j int) bool {
		return strings.Compare(
			aws.ToString(b[i].AttributeName),
			aws.ToString(b[j].AttributeName)) > 0
	})
	return reflect.DeepEqual(a, b)
}

func keySchemaEq(a, b []types.KeySchemaElement) bool {
	if len(a) != len(b) {
		return false
	}
	sort.Slice(a, func(i, j int) bool {
		return strings.Compare(
			aws.ToString(a[i].AttributeName),
			aws.ToString(a[j].AttributeName)) > 0
	})
	sort.Slice(b, func(i, j int) bool {
		return strings.Compare(
			aws.ToString(b[i].AttributeName),
			aws.ToString(b[j].AttributeName)) > 0
	})
	return reflect.DeepEqual(a, b)
}

func secIndexesEq(a []types.GlobalSecondaryIndexDescription, b []types.GlobalSecondaryIndex) bool {
	if len(a) != len(b) {
		return false
	}
	sort.Slice(a, func(i, j int) bool {
		return strings.Compare(
			aws.ToString(a[i].IndexName),
			aws.ToString(a[j].IndexName)) > 0
	})
	sort.Slice(b, func(i, j int) bool {
		return strings.Compare(
			aws.ToString(b[i].IndexName),
			aws.ToString(b[j].IndexName)) > 0
	})
	for i := 0; i < len(a); i++ {
		if !keySchemaEq(a[i].KeySchema, b[i].KeySchema) {
			return false
		}
		if *a[i].IndexName != *b[i].IndexName {
			return false
		}
		if a[i].Projection.ProjectionType != b[i].Projection.ProjectionType {
			return false
		}
	}
	return true
}

func tableActive(
	ctx context.Context,
	ddb AWSDynamoDB,
	tableName string) bool {

	o, err := ddb.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	})
	var notFound *types.ResourceNotFoundException
	if err != nil {
		if errors.As(err, &notFound) {
			return false
		}
		zap.L().Error("failed to describe table", zap.Error(err))
		return false
	}
	if o.Table.TableStatus != types.TableStatusActive {
		fmt.Printf("Table status for %s is %s\n", tableName, o.Table.TableStatus)
	}
	return o.Table.TableStatus == types.TableStatusActive
}

func tableExists(
	ctx context.Context,
	ddb AWSDynamoDB,
	tableName string) bool {

	_, err := ddb.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	})
	var notFound *types.ResourceNotFoundException
	if err != nil {
		if errors.As(err, &notFound) {
			return false
		}
		zap.L().Error("failed to describe table", zap.Error(err))
		return false
	}
	return true
}

func checkTable(
	ctx context.Context,
	ddb AWSDynamoDB,
	tableName string,
	expectedAttrs []types.AttributeDefinition,
	expectedKeySchema []types.KeySchemaElement,
	secIndexes []types.GlobalSecondaryIndex) error {

	t, err := ddb.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	})
	var notFound *types.ResourceNotFoundException
	if err != nil {
		if errors.As(err, &notFound) {
			return apperr.NewNotFound(fmt.Sprintf("table not found: %s", tableName))
		}
		return err
	}
	if !attrDefEq(t.Table.AttributeDefinitions, expectedAttrs) {
		return fmt.Errorf("table already exists with different attributes: %s", tableName)
	}
	if !keySchemaEq(t.Table.KeySchema, expectedKeySchema) {
		return fmt.Errorf("table already exists with different key schema: %s", tableName)
	}
	if !secIndexesEq(t.Table.GlobalSecondaryIndexes, secIndexes) {
		return errors.New("table already exists with different secondary indexes")
	}
	return nil
}

func createTable(
	ctx context.Context,
	ddb AWSDynamoDB,
	tableName string,
	attrs []types.AttributeDefinition,
	keySchema []types.KeySchemaElement,
	secIndexes []types.GlobalSecondaryIndex) error {

	_, err := ddb.CreateTable(ctx, &dynamodb.CreateTableInput{
		AttributeDefinitions:   attrs,
		KeySchema:              keySchema,
		GlobalSecondaryIndexes: secIndexes,
		TableName:              aws.String(tableName),
		BillingMode:            types.BillingModePayPerRequest,
		SSESpecification: &types.SSESpecification{
			Enabled: aws.Bool(true),
			SSEType: types.SSETypeKms,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create table: %s %w", tableName, err)
	}
	return nil
}

func ensureTable(
	ctx context.Context,
	ddb AWSDynamoDB,
	tableName string,
	attrs []types.AttributeDefinition,
	keySchema []types.KeySchemaElement,
	secIndexes []types.GlobalSecondaryIndex) error {

	err := checkTable(ctx, ddb, tableName, attrs, keySchema, secIndexes)
	if apperr.IsNotFound(err) {
		if err := createTable(ctx, ddb, tableName, attrs, keySchema, secIndexes); err != nil {
			return err
		}
		return nil
	}
	return err
}
