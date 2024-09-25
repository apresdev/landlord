package authx

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/apresdev/landlord/internal/apperr"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"

	"github.com/apresdev/landlord/internal/db"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"go.uber.org/zap"

	"github.com/apresdev/landlord/internal/multi"

	"github.com/google/uuid"
)

type Role struct {
	Name   string
	Tenant uuid.UUID
}

func (r Role) String() string {
	return fmt.Sprintf("tenant-%s-%s", r.Tenant, r.Name)
}

type Roles struct {
	Roles []Role
}

func (r Roles) String() string {
	result := make([]string, len(r.Roles))
	for i, role := range r.Roles {
		result[i] = role.String()
	}
	return strings.Join(result, ", ")
}

func ParseRolesFromGroups(groups []string) (Roles, error) {
	result := Roles{Roles: make([]Role, 0)}
	for _, group := range groups {
		role, err := ParseRoleFromGroup(group)
		if err != nil {
			return Roles{}, err
		}
		result.Roles = append(result.Roles, role)
	}
	return result, nil
}

func ParseRoleFromGroup(group string) (Role, error) {
	parts := strings.Split(group, "-")
	if len(parts) != 7 {
		return Role{}, fmt.Errorf("invalid group: %s", group)
	}
	tenantID, err := uuid.Parse(strings.Join(parts[1:6], "-"))
	if err != nil {
		return Role{}, fmt.Errorf("invalid tenant ID: %s", group)
	}
	return Role{Name: parts[6], Tenant: tenantID}, nil
}

func IsAuthorizedForTenant(
	ctx context.Context,
	ddb db.AWSDynamoDB,
	rootTenantID uuid.UUID,
	ten *multi.Tenant,
	roles Roles) bool {

	for _, role := range roles.Roles {
		if role.Tenant == ten.ID {
			return true
		}
	}
	rootTenant, err := db.GetRootTenant(ctx, ddb, rootTenantID)
	if err != nil {
		zap.L().Error("error fetching root tenant", zap.Error(err))
		return false
	}
	for _, role := range roles.Roles {
		if role.Tenant == rootTenant.ID {
			return true
		}
	}
	return false
}

func SetCognitoGroup(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	ten *multi.Tenant,
	user *multi.User) error {

	userGroupName := ten.GroupName(user.Role)
	_, err := cogIdp.AdminAddUserToGroup(ctx,
		&cognitoidentityprovider.AdminAddUserToGroupInput{
			GroupName:  aws.String(userGroupName),
			UserPoolId: aws.String(ten.UserPoolID),
			Username:   aws.String(user.Username),
		})
	if err != nil {
		zap.L().Warn("failed to add user to group",
			zap.Error(err),
			zap.String("user-pool-id", ten.UserPoolID),
			zap.String("tenant-id", ten.ID.String()),
			zap.String("group", userGroupName),
			zap.String("user", user.Username))
	} else {
		zap.L().Info("added user to group",
			zap.String("user-pool-id", ten.UserPoolID),
			zap.String("tenant-id", ten.ID.String()),
			zap.String("group", userGroupName),
			zap.String("user", user.Username))
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
