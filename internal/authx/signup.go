package authx

import (
	"github.com/apresdev/landlord/internal/session"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

func SignUp(userPoolName, email, password string) error {
	up, err := GetUserPoolByName(userPoolName)
	if err != nil {
		return err
	}
	s := session.Must()
	client, err := getSingletonUserPoolClientDescription(up)
	if err != nil {
		return err
	}
	ua := []types.AttributeType{
		{Name: aws.String("name"), Value: aws.String(email)},
	}
	args := cognitoidentityprovider.SignUpInput{
		ClientId:       client.ClientId,
		Password:       aws.String(password),
		Username:       aws.String(email),
		UserAttributes: ua,
	}
	cc := cognitoidentityprovider.NewFromConfig(s.AWS())
	_, err = cc.SignUp(s.Ctx(), &args)
	return err
}

func ConfirmSignUp(userPoolName, email, confirmationCode string) error {
	up, err := GetUserPoolByName(userPoolName)
	if err != nil {
		return err
	}
	client, err := getSingletonUserPoolClientDescription(up)
	if err != nil {
		return err
	}
	args := cognitoidentityprovider.ConfirmSignUpInput{
		ClientId:         client.ClientId,
		Username:         aws.String(email),
		ConfirmationCode: aws.String(confirmationCode),
	}
	s := session.Must()
	cc := cognitoidentityprovider.NewFromConfig(s.AWS())
	_, err = cc.ConfirmSignUp(s.Ctx(), &args)
	return err
}
