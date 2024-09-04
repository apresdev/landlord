package authx

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"

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

func CreateRootTenant(
	ctx context.Context,
	ddb db.AWSDynamoDB,
	cogIdp AWSCognitoIdentityProvider) error {

	rootTenant := multi.NewTenant()
	rootTenant.Name = multi.RootTenantName
	rootTenant.Description = multi.RootTenantDescription
	rootTenant.UserPoolName = os.Getenv("LANDLORD_DEFAULT_USER_POOL")
	up, err := GetUserPoolByNameOrID(ctx, cogIdp, rootTenant.UserPoolName)
	if err != nil {
		return err
	}
	rootTenant.UserPoolID = aws.ToString(up.Id)
	_, err = cogIdp.CreateGroup(ctx, &cognitoidentityprovider.CreateGroupInput{
		GroupName:  aws.String(rootTenant.GroupName(multi.RoleGuest)),
		UserPoolId: aws.String(rootTenant.UserPoolID),
	})
	if err != nil {
		return err
	}
	_, err = cogIdp.CreateGroup(ctx, &cognitoidentityprovider.CreateGroupInput{
		GroupName:  aws.String(rootTenant.GroupName(multi.RoleAdmin)),
		UserPoolId: aws.String(rootTenant.UserPoolID),
	})
	if err != nil {
		return err
	}
	return db.StoreTenant(ctx, ddb, &rootTenant)
}

// CreateTenant creates a new tenant in the system.
func CreateTenant(
	ctx context.Context,
	ddb db.AWSDynamoDB,
	cogIdp AWSCognitoIdentityProvider,
	name string,
	description string) (multi.Tenant, error) {

	if err := multi.ValidateTenantName(name); err != nil {
		return multi.Tenant{}, apperr.NewBadRequestE(err)
	}
	_, err := db.GetTenantByName(ctx, ddb, name)
	if err == nil {
		return multi.Tenant{}, apperr.NewBadRequestE(
			fmt.Errorf("tenant already exists: %s", name))
	}
	if !apperr.IsNotFound(err) {
		return multi.Tenant{}, apperr.NewInternalE(err)
	}
	rootTenant, err := db.GetRootTenant(ctx, ddb)
	if err != nil {
		return multi.Tenant{}, apperr.NewInternalE(err)
	}
	ten := multi.NewTenant()
	ten.Name = strcase.ToKebab(name)
	ten.State = multi.TenantStateActive
	ten.Description = description
	ten.UserPoolName = rootTenant.UserPoolName
	ten.UserPoolID = rootTenant.UserPoolID
	guestGroupName := ten.GroupName(multi.RoleGuest)
	_, err = cogIdp.CreateGroup(ctx, &cognitoidentityprovider.CreateGroupInput{
		GroupName:  aws.String(guestGroupName),
		UserPoolId: aws.String(ten.UserPoolID),
	})
	if err != nil {
		return multi.Tenant{}, apperr.NewInternalE(err)
	}
	if err := db.StoreTenant(ctx, ddb, &ten); err != nil {
		return multi.Tenant{}, apperr.NewInternalE(err)
	}
	return ten, nil
}

func SetTenantOnUser(
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
		},
	}
	_, err := cogIdp.AdminUpdateUserAttributes(ctx, input)
	return err
}

func ApplyTenant(
	ctx context.Context,
	ddb db.AWSDynamoDB,
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
	users, err := db.GetUsersForTenant(ctx, ddb, ten.ID)
	if err != nil {
		return err
	}
	for _, user := range users {
		if err := SetTenantOnUser(ctx, cogIdp, ten, user); err != nil {
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

func GetUsersWithRole(ctx context.Context, ddb db.AWSDynamoDB, tenantID uuid.UUID, role string) ([]*multi.User, error) {
	users, err := db.GetUsersForTenant(ctx, ddb, tenantID)
	if err != nil {
		return nil, err
	}
	var result []*multi.User
	for _, user := range users {
		if user.Role == role {
			result = append(result, user)
		}
	}
	return result, nil
}

func RemoveRoleFromTenant(ctx context.Context, ddb db.AWSDynamoDB, tenantID uuid.UUID, role string) error {
	if role == multi.RoleGuest {
		return errors.New("cannot remove guest role")
	}
	ten, err := db.GetTenantByID(ctx, ddb, tenantID)
	if err != nil {
		return err
	}
	users, err := db.GetUsersForTenant(ctx, ddb, tenantID)
	if err != nil {
		return err
	}
	for _, user := range users {
		if user.Role == role {
			return errors.New("role still in use")
		}
	}
	idx := slices.Index(ten.Roles, role)
	if idx == -1 {
		return errors.New("role not found")
	}
	ten.Roles = append(ten.Roles[:idx], ten.Roles[idx+1:]...)
	return db.StoreTenant(ctx, ddb, ten)
}

func SetTenantState(
	ctx context.Context,
	ddb db.AWSDynamoDB,
	th multi.TenantHandle,
	state multi.TenantState) (*multi.Tenant, error) {

	ten, err := db.ResolveTenant(ctx, ddb, th)
	if err != nil {
		return nil, err
	}
	ten.State = state
	err = db.StoreTenant(ctx, ddb, ten)
	if err != nil {
		return nil, err
	}
	return ten, nil
}
