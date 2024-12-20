package db

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/apresdev/landlord/internal/multi"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

func StoreTenant(ctx context.Context, ddb AWSDynamoDB, ten *multi.Tenant) error {
	if ten.ID == uuid.Nil {
		return apperr.NewInternal("cannot store tenant with nil ID")
	}
	if ten.Name == "" {
		return apperr.NewInternal("cannot store tenant with empty name")
	}
	sort.Strings(ten.Roles)
	sort.Strings(ten.SubTenantRoles)

	if ten.Profile != nil {
		var err error
		ten.RawProfile, err = json.Marshal(ten.Profile)
		if err != nil {
			return apperr.NewInternalE(
				fmt.Errorf("failed to marshal profile: %w", err))
		}
	}

	av, err := attributevalue.MarshalMap(ten)
	if err != nil {
		return err
	}
	av["pk"] = &types.AttributeValueMemberS{Value: GlobalEntityID("*")}
	av["sk"] = &types.AttributeValueMemberS{Value: TenantEntityID(ten.ID.String())}
	av["altName"] = &types.AttributeValueMemberS{Value: TenantEntityID(ten.Name)}
	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(tableName),
	}
	_, err = ddb.PutItem(ctx, input)
	if err != nil {
		return apperr.NewInternalE(
			fmt.Errorf("failed to put item to %s: %w", tableName, err))
	}
	return err
}

func GetTenantByID(ctx context.Context, ddb AWSDynamoDB, id uuid.UUID) (*multi.Tenant, error) {
	kv := make(map[string]types.AttributeValue)
	kv["pk"] = &types.AttributeValueMemberS{Value: GlobalEntityID("*")}
	kv["sk"] = &types.AttributeValueMemberS{Value: TenantEntityID(id.String())}
	input := &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key:       kv,
	}
	result, err := ddb.GetItem(ctx, input)
	if err != nil {
		return nil, apperr.NewInternalE(
			fmt.Errorf("failed to get item from DynamoDB: %w", err))
	}
	if result.Item == nil {
		return nil, apperr.NewNotFound("item not found")
	}
	return unmarshalTenant(result.Item)
}

func GetTenantByName(ctx context.Context, ddb AWSDynamoDB, name string) (*multi.Tenant, error) {
	kv := make(map[string]types.AttributeValue)
	kv[":altName"] = &types.AttributeValueMemberS{Value: TenantEntityID(name)}
	input := &dynamodb.QueryInput{
		TableName:                 aws.String(tableName),
		IndexName:                 aws.String("altNameIndex"),
		KeyConditionExpression:    aws.String("altName = :altName"),
		ExpressionAttributeValues: kv,
	}
	result, err := ddb.Query(ctx, input)
	if err != nil {
		return nil, apperr.NewInternalE(
			fmt.Errorf("failed to get item from DynamoDB: %w", err))
	}
	if len(result.Items) == 0 {
		return nil, apperr.NewNotFound("item not found")
	}
	if len(result.Items) > 1 {
		return nil, apperr.NewInternalE(
			fmt.Errorf("unexpected multiple items found for tenant name %s", name))
	}
	return unmarshalTenant(result.Items[0])
}

func GetRootTenant(
	ctx context.Context,
	ddb AWSDynamoDB,
	rootTenantID uuid.UUID) (*multi.Tenant, error) {

	return GetTenantByID(ctx, ddb, rootTenantID)
}

func GetAllTenants(ctx context.Context, ddb AWSDynamoDB) ([]*multi.Tenant, error) {
	kv := make(map[string]types.AttributeValue)
	kv[":pk"] = &types.AttributeValueMemberS{Value: GlobalEntityID("*")}
	kv[":skPrefix"] = &types.AttributeValueMemberS{Value: TenantEntityID("")}
	input := &dynamodb.QueryInput{
		TableName:                 aws.String(tableName),
		KeyConditionExpression:    aws.String("pk = :pk AND begins_with(sk, :skPrefix)"),
		ExpressionAttributeValues: kv,
	}
	result, err := ddb.Query(ctx, input)
	if err != nil {
		return nil, apperr.NewInternalE(
			fmt.Errorf("failed to scan DynamoDB: %w", err))
	}
	var tenants []*multi.Tenant
	for _, item := range result.Items {
		tenant, err := unmarshalTenant(item)
		if err != nil {
			return nil, err
		}
		tenants = append(tenants, tenant)
	}
	// Sort tenants by name, with root tenant always first
	sort.Slice(tenants, func(i, j int) bool {
		if tenants[i].Name == multi.RootTenantName {
			return true
		}
		if tenants[j].Name == multi.RootTenantName {
			return false
		}
		return tenants[i].Name < tenants[j].Name
	})
	return tenants, nil
}

func ResolveTenant(ctx context.Context, ddb AWSDynamoDB, h multi.TenantHandle) (*multi.Tenant, error) {
	if !h.IsSet() {
		return nil, apperr.NewBadRequest("tenant ID or tenant name must be set")
	}
	if h.ID.Version() != 0 {
		return GetTenantByID(ctx, ddb, h.ID)
	}
	return GetTenantByName(ctx, ddb, h.Name)
}

func parseEntityUUID(v types.AttributeValue, entityPrefix string) (uuid.UUID, error) {
	if v == nil {
		return uuid.Nil, apperr.NewInternal("missing entity ID")
	}
	s, ok := v.(*types.AttributeValueMemberS)
	if !ok {
		return uuid.Nil, apperr.NewInternal("entity ID is not a string")
	}
	idStr, err := StripEntityID(entityPrefix, s.Value)
	if err != nil {
		return uuid.Nil, apperr.NewInternalE(
			fmt.Errorf("failed to strip entity ID: %w", err))
	}
	return uuid.Parse(idStr)
}

func parseEntityName(v types.AttributeValue, entityPrefix string) (string, error) {
	s, err := resolveStr(v)
	if err != nil {
		return "", err
	}
	return StripEntityID(entityPrefix, s)
}

func unmarshalTenant(kv map[string]types.AttributeValue) (*multi.Tenant, error) {
	var tenant multi.Tenant
	err := attributevalue.UnmarshalMap(kv, &tenant)
	if err != nil {
		return nil, apperr.NewInternalE(
			fmt.Errorf("failed to unmarshal DynamoDB item to Tenant: %w", err))
	}
	tenant.ID, err = parseEntityUUID(kv["sk"], EntityNameTenant)
	if err != nil {
		return nil, apperr.NewInternalE(
			fmt.Errorf("failed to parse tenant ID: %w", err))
	}
	tenant.Name, err = parseEntityName(kv["altName"], EntityNameTenant)
	if err != nil {
		return nil, apperr.NewInternalE(
			fmt.Errorf("failed to strip entity ID: %w", err))
	}
	return &tenant, nil
}
