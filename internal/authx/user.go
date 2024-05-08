package authx

import (
	"context"

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
