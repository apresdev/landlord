package authx

import (
	"github.com/apresdev/landlord/internal/session"
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

func DeleteUser(userPool, email string) error {
	up, err := GetUserPoolByNameOrID(userPool)
	if err != nil {
		return err
	}
	s := session.Must()
	cc := cognitoidentityprovider.NewFromConfig(s.AWS())
	_, err = cc.AdminDeleteUser(s.Ctx(), &cognitoidentityprovider.AdminDeleteUserInput{
		UserPoolId: up.Id,
		Username:   &email,
	})
	return err
}
