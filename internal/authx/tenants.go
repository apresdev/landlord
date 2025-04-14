package authx

import (
	"context"
	"errors"
	"fmt"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/apresdev/landlord/internal/db"
	"github.com/apresdev/landlord/internal/multi"

	"go.uber.org/zap"

	"github.com/google/uuid"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	cogidptypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/ettle/strcase"
)

func MustRootTenant(
	ctx context.Context,
	dbc *db.Context,
	cogIdp AWSCognitoIdentityProvider,
	userPoolName string) uuid.UUID {

	rootTenant, err := db.GetTenantByName(ctx, dbc, multi.RootTenantName)
	if err != nil && !apperr.IsNotFound(err) {
		zap.L().Fatal("unexpected error getting root tenant", zap.Error(err))
	}
	if rootTenant != nil {
		return rootTenant.ID
	}

	rootTenant = multi.NewTenant(multi.RootTenantName)
	rootTenant.Description = multi.RootTenantDescription
	rootTenant.UserPoolName = userPoolName
	up, err := GetUserPoolByNameOrID(ctx, cogIdp, rootTenant.UserPoolName)
	if err != nil {
		zap.L().Fatal("error getting user pool for root tenant",
			zap.String("userPoolName", rootTenant.UserPoolName),
			zap.Error(err))
	}
	rootTenant.UserPoolID = aws.ToString(up.Id)
	for _, role := range multi.BuiltInRoles {
		_, err = cogIdp.CreateGroup(ctx, &cognitoidentityprovider.CreateGroupInput{
			GroupName:  aws.String(rootTenant.GroupName(role)),
			UserPoolId: aws.String(rootTenant.UserPoolID),
		})
		var groupExists *cogidptypes.GroupExistsException
		if err != nil && !errors.As(err, &groupExists) {
			zap.L().Fatal("error creating group for root tenant", zap.Error(err))
		}
	}
	rootTenant.Roles = multi.BuiltInRoles
	err = db.StoreTenant(ctx, dbc, rootTenant)
	if err != nil {
		zap.L().Fatal("error storing root tenant", zap.Error(err))
	} else {
		zap.L().Info("created root tenant",
			zap.String("id", rootTenant.ID.String()))
	}
	return rootTenant.ID
}

// CreateTenant creates a new tenant in the system.
func CreateTenant(
	ctx context.Context,
	dbc *db.Context,
	rootTenantID uuid.UUID,
	cogIdp AWSCognitoIdentityProvider,
	name string,
	description string) (*multi.Tenant, error) {

	if err := multi.ValidateTenantName(name); err != nil {
		return nil, apperr.NewBadRequestE(err)
	}
	_, err := db.GetTenantByName(ctx, dbc, name)
	if err == nil {
		return nil, apperr.NewBadRequestE(
			fmt.Errorf("tenant already exists: %s", name))
	}
	if !apperr.IsNotFound(err) {
		return nil, apperr.NewInternalE(err)
	}
	rootTenant, err := db.GetRootTenant(ctx, dbc, rootTenantID)
	if err != nil {
		return nil, apperr.NewInternalE(err)
	}
	ten := multi.NewTenant(strcase.ToKebab(name))
	ten.State = multi.TenantStateActive
	ten.Description = description
	ten.UserPoolName = rootTenant.UserPoolName
	ten.UserPoolID = rootTenant.UserPoolID
	for _, role := range rootTenant.SubTenantRoles {
		_, err = cogIdp.CreateGroup(ctx, &cognitoidentityprovider.CreateGroupInput{
			GroupName:  aws.String(ten.GroupName(role)),
			UserPoolId: aws.String(ten.UserPoolID),
		})
		if err != nil {
			return nil, apperr.NewInternalE(err)
		}
	}
	ten.Roles = rootTenant.SubTenantRoles
	if err := db.StoreTenant(ctx, dbc, ten); err != nil {
		return nil, apperr.NewInternalE(err)
	}
	return ten, nil
}

func setCustomClaimsOnUser(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	ten *multi.Tenant,
	user *multi.User) error {

	input := &cognitoidentityprovider.AdminUpdateUserAttributesInput{
		UserPoolId: aws.String(ten.UserPoolID),
		Username:   aws.String(user.Username),
		UserAttributes: []cogidptypes.AttributeType{
			{
				Name:  aws.String(CustomTenantClaim),
				Value: aws.String(ten.ID.String()),
			},
			{
				Name:  aws.String(CustomUserClaim),
				Value: aws.String(user.ID.String()),
			},
		},
	}
	_, err := cogIdp.AdminUpdateUserAttributes(ctx, input)
	return err
}

func ApplyTenant(
	ctx context.Context,
	dbc *db.Context,
	cogIdp AWSCognitoIdentityProvider,
	ten *multi.Tenant) error {

	for _, role := range ten.Roles {
		groupName := ten.GroupName(role)
		o, err := cogIdp.GetGroup(ctx, &cognitoidentityprovider.GetGroupInput{
			GroupName:  aws.String(groupName),
			UserPoolId: aws.String(ten.UserPoolID),
		})
		if o != nil && err == nil {
			zap.L().Info("group already exists", zap.String("name", groupName))
			continue
		}
		_, err = cogIdp.CreateGroup(ctx, &cognitoidentityprovider.CreateGroupInput{
			GroupName:  aws.String(groupName),
			UserPoolId: aws.String(ten.UserPoolID),
		})
		if err != nil {
			return err
		}
		zap.L().Info("created group", zap.String("name", groupName))
	}
	users, err := db.GetUsersForTenant(ctx, dbc, ten.ID)
	if err != nil {
		return err
	}
	for _, user := range users {
		if err := setCustomClaimsOnUser(ctx, cogIdp, ten, user); err != nil {
			return err
		}
		zap.L().Info("updated user attributes",
			zap.String("tenant", ten.ID.String()),
			zap.String("user", user.Username))
		userGroupName := ten.GroupName(user.Role)
		_, err := cogIdp.AdminAddUserToGroup(ctx, &cognitoidentityprovider.AdminAddUserToGroupInput{
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
	}
	return nil
}

func SetTenantState(
	ctx context.Context,
	dbc *db.Context,
	ten *multi.Tenant,
	state multi.TenantState) (*multi.Tenant, error) {

	if ten.IsRoot() {
		return nil, apperr.NewBadRequest("cannot change the root tenant's state")
	}
	ten.State = state
	err := db.StoreTenant(ctx, dbc, ten)
	if err != nil {
		return nil, err
	}
	return ten, nil
}
