package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/apresdev/landlord/internal/multi"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

const (
	tenantsTable = "landlord-tenants"
	idAttr       = "id"
	nameAttr     = "name"
)

func initTenants(ctx context.Context, ddb AWSDynamoDB) error {
	attrs := []types.AttributeDefinition{
		{
			AttributeName: aws.String(idAttr),
			AttributeType: types.ScalarAttributeTypeB,
		},
		{
			AttributeName: aws.String(nameAttr),
			AttributeType: types.ScalarAttributeTypeS,
		},
	}
	keySchema := []types.KeySchemaElement{
		{
			AttributeName: aws.String(idAttr),
			KeyType:       types.KeyTypeHash,
		},
	}
	secIndexes := []types.GlobalSecondaryIndex{
		{
			IndexName: aws.String("sec_index_" + nameAttr),
			KeySchema: []types.KeySchemaElement{
				{
					AttributeName: aws.String(nameAttr),
					KeyType:       types.KeyTypeHash,
				},
			},
			Projection: &types.Projection{
				ProjectionType: types.ProjectionTypeAll,
			},
		},
	}
	return ensureTable(ctx, ddb, tenantsTable, attrs, keySchema, secIndexes)
}

func StoreTenant(ctx context.Context, ddb AWSDynamoDB, ten *multi.Tenant) error {
	av, err := attributevalue.MarshalMap(ten)
	if err != nil {
		return err
	}
	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(tenantsTable),
	}
	_, err = ddb.PutItem(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to put item to %s: %w", tenantsTable, err)
	}
	return err
}

func GetTenantByID(ctx context.Context, ddb AWSDynamoDB, id uuid.UUID) (*multi.Tenant, error) {
	b, err := id.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal UUID to binary: %w", err)
	}
	input := &dynamodb.GetItemInput{
		TableName: aws.String(tenantsTable),
		Key: map[string]types.AttributeValue{
			idAttr: &types.AttributeValueMemberB{Value: b},
		},
	}
	result, err := ddb.GetItem(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get item from DynamoDB: %w", err)
	}
	if result.Item == nil {
		return nil, apperr.NewNotFound("item not found")
	}
	var tenant multi.Tenant
	err = attributevalue.UnmarshalMap(result.Item, &tenant)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal DynamoDB item to Tenant: %w", err)
	}
	return &tenant, nil
}

func GetTenantByName(ctx context.Context, ddb AWSDynamoDB, name string) (*multi.Tenant, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(tenantsTable),
		IndexName:              aws.String("sec_index_name"),
		KeyConditionExpression: aws.String("#name = :n"),
		ExpressionAttributeNames: map[string]string{
			"#name": nameAttr,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":n": &types.AttributeValueMemberS{Value: name},
		},
	}
	o, err := ddb.Query(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to query DynamoDB: %w", err)
	}
	if o.Items == nil || len(o.Items) == 0 {
		return nil, apperr.NewNotFound("item not found")
	}
	if len(o.Items) > 1 {
		return nil, apperr.NewInternal("multiple items found")
	}
	var tenant multi.Tenant
	err = attributevalue.UnmarshalMap(o.Items[0], &tenant)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal DynamoDB item to Tenant: %w", err)
	}
	return &tenant, nil
}

func GetRootTenant(ctx context.Context, ddb AWSDynamoDB) (*multi.Tenant, error) {
	return GetTenantByName(ctx, ddb, multi.RootTenantName)
}

func GetAllTenants(ctx context.Context, ddb AWSDynamoDB) ([]*multi.Tenant, error) {
	input := &dynamodb.ScanInput{
		TableName: aws.String(tenantsTable),
	}
	result, err := ddb.Scan(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to scan DynamoDB: %w", err)
	}
	var tenants []*multi.Tenant
	for _, item := range result.Items {
		var tenant multi.Tenant
		err = attributevalue.UnmarshalMap(item, &tenant)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal DynamoDB item to Tenant: %w", err)
		}
		tenants = append(tenants, &tenant)
	}
	return tenants, nil
}

func ResolveTenant(ctx context.Context, ddb AWSDynamoDB, h multi.TenantHandle) (*multi.Tenant, error) {
	if !h.IsSet() {
		return nil, errors.New("tenant ID or tenant name must be set")
	}
	if h.ID.Version() != 0 {
		return GetTenantByID(ctx, ddb, h.ID)
	}
	return GetTenantByName(ctx, ddb, h.Name)
}
