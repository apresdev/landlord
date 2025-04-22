package authx

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/apresdev/landlord/internal/apperr"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"go.uber.org/zap"

	"github.com/apresdev/landlord/internal/multi"

	"github.com/google/uuid"
)

func ParseRolesFromGroups(groups []string) (*multi.Roles, error) {
	result := multi.NewRoles()
	for _, group := range groups {
		role, err := ParseRoleFromGroup(group)
		if err != nil {
			return nil, err
		}
		result.Roles = append(result.Roles, role)
	}
	return result, nil
}

func ParseRoleFromGroup(group string) (*multi.Role, error) {
	parts := strings.Split(group, "-")
	if len(parts) != 7 {
		return nil, fmt.Errorf("invalid group: %s", group)
	}
	tenantID, err := uuid.Parse(strings.Join(parts[1:6], "-"))
	if err != nil {
		return nil, fmt.Errorf("invalid tenant ID: %s", group)
	}
	return multi.NewRole(parts[6], tenantID), nil
}

func SetCognitoGroup(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	ten *multi.Tenant,
	username string,
	role string) error {

	groups, err := cogIdp.AdminListGroupsForUser(ctx,
		&cognitoidentityprovider.AdminListGroupsForUserInput{
			Username:   aws.String(username),
			UserPoolId: aws.String(ten.UserPoolID),
		})
	if err != nil {
		zap.L().Error("failed to list groups for user",
			zap.Error(err),
			zap.String("username", username),
			zap.String("userPoolID", ten.UserPoolID),
			zap.String("tenantID", ten.ID.String()))
		return err
	}

	newGroupName := ten.GroupName(role)
	groupsToRemove := make([]string, 0)
	for _, group := range groups.Groups {
		if aws.ToString(group.GroupName) == newGroupName {
			zap.L().Info("user already in group",
				zap.String("username", username),
				zap.String("groupName", newGroupName),
				zap.String("userPoolID", ten.UserPoolID),
				zap.String("tenantID", ten.ID.String()))
			return nil
		}
		groupsToRemove = append(groupsToRemove, aws.ToString(group.GroupName))
	}

	_, err = cogIdp.AdminAddUserToGroup(ctx,
		&cognitoidentityprovider.AdminAddUserToGroupInput{
			GroupName:  aws.String(newGroupName),
			UserPoolId: aws.String(ten.UserPoolID),
			Username:   aws.String(username),
		})
	if err != nil {
		zap.L().Warn("failed to add user to group",
			zap.Error(err),
			zap.String("username", username),
			zap.String("groupName", newGroupName),
			zap.String("userPoolID", ten.UserPoolID),
			zap.String("tenantID", ten.ID.String()))
	} else {
		zap.L().Info("added user to group",
			zap.String("username", username),
			zap.String("groupName", newGroupName),
			zap.String("userPoolID", ten.UserPoolID),
			zap.String("tenantID", ten.ID.String()))
	}

	for _, groupToRemove := range groupsToRemove {
		_, err = cogIdp.AdminRemoveUserFromGroup(ctx,
			&cognitoidentityprovider.AdminRemoveUserFromGroupInput{
				GroupName:  aws.String(groupToRemove),
				UserPoolId: aws.String(ten.UserPoolID),
				Username:   aws.String(username),
			})
		if err != nil {
			zap.L().Warn("failed to remove user from group",
				zap.Error(err),
				zap.String("username", username),
				zap.String("groupName", groupToRemove),
				zap.String("userPoolID", ten.UserPoolID),
				zap.String("tenantID", ten.ID.String()))
		} else {
			zap.L().Info("removed user from group",
				zap.String("username", username),
				zap.String("groupName", groupToRemove),
				zap.String("userPoolID", ten.UserPoolID),
				zap.String("tenantID", ten.ID.String()))
		}
	}

	return err
}

func CreateGroupForTenant(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	ten *multi.Tenant,
	role string) error {

	groupName := ten.GroupName(role)
	_, err := cogIdp.CreateGroup(ctx,
		&cognitoidentityprovider.CreateGroupInput{
			GroupName:  aws.String(groupName),
			UserPoolId: aws.String(ten.UserPoolID),
		})
	var groupExists *types.GroupExistsException
	if err != nil && errors.As(err, &groupExists) {
		return apperr.NewAlreadyExistsE(err)
	}
	if err != nil {
		zap.L().Error("failed to create group",
			zap.Error(err),
			zap.String("user-pool-id", ten.UserPoolID),
			zap.String("tenant-id", ten.ID.String()),
			zap.String("group", groupName))
		return err
	}
	zap.L().Info("created group",
		zap.String("user-pool-id", ten.UserPoolID),
		zap.String("tenant-id", ten.ID.String()),
		zap.String("group", groupName))
	return nil
}
