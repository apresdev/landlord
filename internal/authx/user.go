package authx

import (
	"context"

	"github.com/sethvargo/go-password/password"

	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

func UserEmailAttribute(u types.UserType) string {
	for _, a := range u.Attributes {
		if *a.Name == "email" {
			return *a.Value
		}
	}
	return ""
}

func CreateUser(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	userPool, email string) error {

	pwd, err := password.Generate(16, 4, 4, false, false)
	if err != nil {
		return err
	}

	up, err := GetUserPoolByNameOrID(ctx, cogIdp, userPool)
	if err != nil {
		return err
	}
	_, err = cogIdp.AdminCreateUser(ctx,
		&cognitoidentityprovider.AdminCreateUserInput{
			UserPoolId:        up.Id,
			Username:          &email,
			TemporaryPassword: &pwd,
		})
	return err
}

func DeleteUser(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	userPool, email string) error {

	up, err := GetUserPoolByNameOrID(ctx, cogIdp, userPool)
	if err != nil {
		return err
	}
	_, err = cogIdp.AdminDeleteUser(ctx,
		&cognitoidentityprovider.AdminDeleteUserInput{
			UserPoolId: up.Id,
			Username:   &email,
		})
	return err
}
