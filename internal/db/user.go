package db

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/apresdev/landlord/internal/multi"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

// StoreUser stores the given user in the database, creating a new item if
// it does not already exist.
func StoreUser(ctx context.Context, ddb AWSDynamoDB, user *multi.User) error {
	if user.TenantID == uuid.Nil {
		return apperr.NewInternal("cannot store user with nil tenant ID")
	}
	if user.ID == uuid.Nil {
		return apperr.NewInternal("cannot store user with nil ID")
	}
	if user.Username == "" {
		return apperr.NewInternal("cannot store user with empty username")
	}
	if user.Role == "" {
		return apperr.NewInternal("cannot store user with empty role")
	}
	if user.Profile != nil {
		var err error
		user.RawProfile, err = json.Marshal(user.Profile)
		if err != nil {
			return apperr.NewInternalE(
				fmt.Errorf("failed to marshal user profile: %w", err))
		}
	}

	av, err := attributevalue.MarshalMap(user)
	if err != nil {
		return err
	}
	av["pk"] = &types.AttributeValueMemberS{Value: TenantEntityID(user.TenantID.String())}
	av["sk"] = &types.AttributeValueMemberS{Value: UserEntityID(user.ID.String())}
	av["altName"] = &types.AttributeValueMemberS{Value: UserEntityID(user.Username)}
	_, err = ddb.PutItem(ctx, &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(tableName),
	})
	if err != nil {
		return apperr.NewInternalE(
			fmt.Errorf("failed to put item to %s: %w", tableName, err))
	}
	return nil
}

// DeleteUser removes the user with the given username.
func DeleteUser(ctx context.Context, ddb AWSDynamoDB, user *multi.User) error {
	kv := make(map[string]types.AttributeValue)
	kv["pk"] = &types.AttributeValueMemberS{Value: TenantEntityID(user.TenantID.String())}
	kv["sk"] = &types.AttributeValueMemberS{Value: UserEntityID(user.ID.String())}
	input := &dynamodb.DeleteItemInput{
		Key:          kv,
		TableName:    aws.String(tableName),
		ReturnValues: types.ReturnValueAllOld,
	}
	o, err := ddb.DeleteItem(ctx, input)
	if err != nil {
		return apperr.NewInternalE(
			fmt.Errorf("failed to delete item: %w", err))
	}
	if o.Attributes == nil {
		return apperr.NewNotFound("user not found: " + user.Username)
	}
	zap.L().Info("deleted user",
		zap.String("tenant", user.TenantID.String()),
		zap.String("username", user.Username),
		zap.String("userID", user.ID.String()),
		zap.Any("output", o))
	return nil
}

// GetUserByName returns the user with the given username.
func GetUserByName(ctx context.Context, ddb AWSDynamoDB, username string) (*multi.User, error) {
	kv := make(map[string]types.AttributeValue)
	kv[":altName"] = &types.AttributeValueMemberS{Value: UserEntityID(username)}
	input := &dynamodb.QueryInput{
		TableName:                 aws.String(tableName),
		IndexName:                 aws.String("altNameIndex"),
		KeyConditionExpression:    aws.String("altName = :altName"),
		ExpressionAttributeValues: kv,
	}
	o1, err := ddb.Query(ctx, input)
	if err != nil {
		return nil, apperr.NewInternalE(
			fmt.Errorf("failed to get item from DynamoDB: %w", err))
	}
	if len(o1.Items) == 0 {
		return nil, apperr.NewNotFound("user not found")
	}
	if len(o1.Items) > 1 {
		return nil, apperr.NewInternalE(
			fmt.Errorf("found multiple users with username: %s", username))
	}
	return unmarshalUser(o1.Items[0])
}

// GetUserInTenantByID returns a user belonging to a tenant.
func GetUserInTenantByID(
	ctx context.Context, ddb AWSDynamoDB, tenantID uuid.UUID, userID uuid.UUID) (
	*multi.User, error) {

	kv := make(map[string]types.AttributeValue)
	kv["pk"] = &types.AttributeValueMemberS{Value: TenantEntityID(tenantID.String())}
	kv["sk"] = &types.AttributeValueMemberS{Value: UserEntityID(userID.String())}
	result, err := ddb.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key:       kv,
	})
	if err != nil {
		return nil, apperr.NewInternalE(
			fmt.Errorf("failed to get item from DynamoDB: %w", err))
	}
	if result.Item == nil {
		zap.L().Warn("item not found in GetUserInTenantByID",
			zap.String("tenantID", tenantID.String()),
			zap.String("userID", userID.String()))
		return nil, apperr.NewNotFound("item not found")
	}
	return unmarshalUser(result.Item)
}

// GetUsersForTenant returns all users for the given tenant.
func GetUsersForTenant(
	ctx context.Context, ddb AWSDynamoDB, tenantID uuid.UUID) (
	[]*multi.User, error) {

	kv := make(map[string]types.AttributeValue)
	kv[":pk"] = &types.AttributeValueMemberS{Value: TenantEntityID(tenantID.String())}
	kv[":skPrefix"] = &types.AttributeValueMemberS{Value: UserEntityID("")}
	input := &dynamodb.QueryInput{
		TableName:                 aws.String(tableName),
		KeyConditionExpression:    aws.String("pk = :pk AND begins_with(sk, :skPrefix)"),
		ExpressionAttributeValues: kv,
	}
	result, err := ddb.Query(ctx, input)
	if err != nil {
		return nil, apperr.NewInternalE(
			fmt.Errorf("failed to query DynamoDB: %w", err))
	}
	var users []*multi.User
	for _, item := range result.Items {
		user, err := unmarshalUser(item)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

// GetUserAndTenant returns the user and tenant for the given username.
func GetUserAndTenant(
	ctx context.Context, ddb AWSDynamoDB, username string) (
	*multi.User, *multi.Tenant, error) {

	user, err := GetUserByName(ctx, ddb, username)
	if err != nil {
		return nil, nil, err
	}
	ten, err := GetTenantByID(ctx, ddb, user.TenantID)
	if err != nil {
		return nil, nil, err
	}
	return user, ten, nil
}

func SetUserStatus(
	ctx context.Context,
	ddb AWSDynamoDB,
	user *multi.User,
	targetStatus multi.UserStatus) (*multi.User, error) {

	if user.Status == targetStatus {
		return user, nil
	}
	if user.Status == multi.UserStatusInvited {
		return nil, apperr.NewBadRequest(
			"cannot set status to " + targetStatus.String() + " for invited user")
	}
	user.Status = targetStatus
	if err := StoreUser(ctx, ddb, user); err != nil {
		return nil, err
	}
	return user, nil
}

func unmarshalUser(item map[string]types.AttributeValue) (*multi.User, error) {
	var user multi.User
	if err := attributevalue.UnmarshalMap(item, &user); err != nil {
		return nil, apperr.NewInternalE(
			fmt.Errorf("failed to unmarshal DynamoDB item to User: %w", err))
	}
	var err error
	user.TenantID, err = parseEntityUUID(item["pk"], EntityNameTenant)
	if err != nil {
		return nil, apperr.NewInternalE(
			fmt.Errorf("failed to parse user's tenant ID: %w", err))
	}
	user.ID, err = parseEntityUUID(item["sk"], EntityNameUser)
	if err != nil {
		return nil, apperr.NewInternalE(
			fmt.Errorf("failed to parse user ID: %w", err))
	}
	user.Username, err = parseEntityName(item["altName"], EntityNameUser)
	if err != nil {
		return nil, apperr.NewInternalE(
			fmt.Errorf("failed to strip entity ID: %w", err))
	}
	return &user, nil
}
