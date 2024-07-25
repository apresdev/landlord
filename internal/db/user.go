package db

import (
	"context"
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
	usersTable   = "landlord-users"
	usernameAttr = "username"
	tenantAttr   = "tenant"
)

func initUsers(ctx context.Context, dynamo AWSDynamoDB) error {
	attrs := []types.AttributeDefinition{
		{
			AttributeName: aws.String(tenantAttr),
			AttributeType: types.ScalarAttributeTypeB,
		},
		{
			AttributeName: aws.String(usernameAttr),
			AttributeType: types.ScalarAttributeTypeS,
		},
	}
	keySchema := []types.KeySchemaElement{
		{
			AttributeName: aws.String(tenantAttr),
			KeyType:       types.KeyTypeHash,
		},
		{
			AttributeName: aws.String(usernameAttr),
			KeyType:       types.KeyTypeRange,
		},
	}
	secIndexes := []types.GlobalSecondaryIndex{
		{
			IndexName: aws.String("sec_index_" + usernameAttr),
			KeySchema: []types.KeySchemaElement{
				{
					AttributeName: aws.String(usernameAttr),
					KeyType:       types.KeyTypeHash,
				},
			},
			Projection: &types.Projection{
				ProjectionType: types.ProjectionTypeAll,
			},
		},
	}
	return ensureTable(ctx, dynamo, usersTable, attrs, keySchema, secIndexes)
}

func StoreUser(ctx context.Context, ddb AWSDynamoDB, ten *multi.User) error {
	av, err := attributevalue.MarshalMap(ten)
	if err != nil {
		return err
	}
	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(usersTable),
	}
	_, err = ddb.PutItem(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to put item to %s: %w", usersTable, err)
	}
	return err
}

func DeleteUser(
	ctx context.Context,
	ddb AWSDynamoDB,
	tenantID uuid.UUID,
	username string) error {

	b, err := tenantID.MarshalBinary()
	if err != nil {
		return err
	}
	input := &dynamodb.DeleteItemInput{
		Key: map[string]types.AttributeValue{
			tenantAttr:   &types.AttributeValueMemberB{Value: b},
			usernameAttr: &types.AttributeValueMemberS{Value: username},
		},
		TableName: aws.String(usersTable),
	}
	_, err = ddb.DeleteItem(ctx, input)
	return err
}

func GetUser(ctx context.Context, ddb AWSDynamoDB, username string) (*multi.User, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(usersTable),
		IndexName:              aws.String("sec_index_" + usernameAttr),
		KeyConditionExpression: aws.String("#username = :username"),
		ExpressionAttributeNames: map[string]string{
			"#username": usernameAttr,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":username": &types.AttributeValueMemberS{Value: username},
		},
	}
	q, err := ddb.Query(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to query DynamoDB: %w", err)
	}
	if len(q.Items) == 0 {
		return nil, apperr.NewNotFound("username not found: " + username)
	}
	if len(q.Items) > 1 {
		return nil, fmt.Errorf("multiple users found with username: %s", username)
	}
	var result multi.User
	if err = attributevalue.UnmarshalMap(q.Items[0], &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal DynamoDB item to User: %w", err)
	}
	return &result, nil
}

func GetUsersForTenant(ctx context.Context, ddb AWSDynamoDB, tenantID uuid.UUID) ([]*multi.User, error) {
	b, err := tenantID.MarshalBinary()
	if err != nil {
		return nil, err
	}
	input := &dynamodb.QueryInput{
		TableName:              aws.String(usersTable),
		KeyConditionExpression: aws.String("#tenant = :tenant"),
		ExpressionAttributeNames: map[string]string{
			"#tenant": tenantAttr,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":tenant": &types.AttributeValueMemberB{Value: b},
		},
	}
	result, err := ddb.Query(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to query DynamoDB: %w", err)
	}
	var users []*multi.User
	if err = attributevalue.UnmarshalListOfMaps(result.Items, &users); err != nil {
		return nil, fmt.Errorf("failed to unmarshal DynamoDB item to User: %w", err)
	}
	return users, nil
}

func GetUserAndTenant(ctx context.Context, ddb AWSDynamoDB, username string) (*multi.User, *multi.Tenant, error) {
	user, err := GetUser(ctx, ddb, username)
	if err != nil {
		return nil, nil, err
	}
	ten, err := GetTenantByID(ctx, ddb, user.TenantID)
	if err != nil {
		return nil, nil, err
	}
	return user, ten, nil
}
