package authx

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

func SignUp(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	userPool, email, password string) error {

	up, err := GetUserPoolByNameOrID(ctx, cogIdp, userPool)
	if err != nil {
		return err
	}
	client, err := getSingletonUserPoolClientDescription(ctx, cogIdp, up)
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
	_, err = cogIdp.SignUp(ctx, &args)
	return err
}

func ConfirmSignUp(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	userPool, email, confirmationCode string) error {

	up, err := GetUserPoolByNameOrID(ctx, cogIdp, userPool)
	if err != nil {
		return err
	}
	client, err := getSingletonUserPoolClientDescription(ctx, cogIdp, up)
	if err != nil {
		return err
	}
	args := cognitoidentityprovider.ConfirmSignUpInput{
		ClientId:         client.ClientId,
		Username:         aws.String(email),
		ConfirmationCode: aws.String(confirmationCode),
	}
	_, err = cogIdp.ConfirmSignUp(ctx, &args)
	return err
}

func AdminConfirmSignUp(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	userPool, email string) error {

	up, err := GetUserPoolByNameOrID(ctx, cogIdp, userPool)
	if err != nil {
		return err
	}
	args := cognitoidentityprovider.AdminConfirmSignUpInput{
		UserPoolId: up.Id,
		Username:   aws.String(email),
	}
	_, err = cogIdp.AdminConfirmSignUp(ctx, &args)
	return err
}
