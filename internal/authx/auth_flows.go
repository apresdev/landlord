package authx

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
)

type UserPasswordAuthArgs struct {
	Region   string
	UserPool string
	Email    string
	Password string
}

func UserPasswordAuth(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	validator TokenValidator,
	args UserPasswordAuthArgs) (*Tokens, error) {

	up, err := GetUserPoolByNameOrID(ctx, cogIdp, args.UserPool)
	if err != nil {
		return nil, err
	}
	client, err := getSingletonUserPoolClientDescription(ctx, cogIdp, up)
	if err != nil {
		return nil, err
	}

	o, err := cogIdp.InitiateAuth(ctx, &cognitoidentityprovider.InitiateAuthInput{
		AuthFlow: "USER_PASSWORD_AUTH",
		ClientId: client.ClientId,
		AuthParameters: map[string]string{
			"USERNAME": args.Email, "PASSWORD": args.Password},
	})
	if err != nil {
		return nil, err
	}
	_, err = validator.ValidateToken(
		ctx, args.Region, up, *o.AuthenticationResult.AccessToken)
	if err != nil {
		return nil, err
	}
	_, err = validator.ValidateToken(
		ctx, args.Region, up, *o.AuthenticationResult.IdToken)
	if err != nil {
		return nil, err
	}
	return &Tokens{
		AccessToken:  *o.AuthenticationResult.AccessToken,
		IdToken:      *o.AuthenticationResult.IdToken,
		RefreshToken: *o.AuthenticationResult.RefreshToken,
	}, nil
}

type RefreshAuthArgs struct {
	Region       string
	UserPool     string
	RefreshToken string
}

func RefreshAuth(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	validator TokenValidator,
	args RefreshAuthArgs) (*Tokens, error) {

	up, err := GetUserPoolByNameOrID(ctx, cogIdp, args.UserPool)
	if err != nil {
		return nil, err
	}
	client, err := getSingletonUserPoolClientDescription(ctx, cogIdp, up)
	if err != nil {
		return nil, err
	}
	o, err := cogIdp.InitiateAuth(ctx, &cognitoidentityprovider.InitiateAuthInput{
		AuthFlow:       "REFRESH_TOKEN_AUTH",
		ClientId:       client.ClientId,
		AuthParameters: map[string]string{"REFRESH_TOKEN": args.RefreshToken},
	})
	if err != nil {
		return nil, err
	}
	tokens := Tokens{
		AccessToken:  *o.AuthenticationResult.AccessToken,
		IdToken:      *o.AuthenticationResult.IdToken,
		RefreshToken: args.RefreshToken,
	}
	_, err = validator.ValidateToken(ctx, args.Region, up, tokens.AccessToken)
	if err != nil {
		return nil, err
	}
	_, err = validator.ValidateToken(ctx, args.Region, up, tokens.IdToken)
	if err != nil {
		return nil, err
	}
	return &tokens, nil
}
