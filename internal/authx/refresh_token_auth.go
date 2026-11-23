package authx

import (
	"context"
	"net/url"

	"github.com/apresdev/landlord/internal/jwx"

	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
)

func RefreshTokenAuth(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	validator jwx.TokenValidator,
	validationURL *url.URL,
	up *types.UserPoolType,
	refreshToken string,
	preferredClient string) (*jwx.RawTokens, error) {

	client, err := selectUserPoolClientDescription(
		ctx, cogIdp, aws.ToString(up.Id), preferredClient)
	if err != nil {
		return nil, err
	}
	o, err := cogIdp.InitiateAuth(ctx, &cognitoidentityprovider.InitiateAuthInput{
		AuthFlow:       "REFRESH_TOKEN_AUTH",
		ClientId:       client.ClientId,
		AuthParameters: map[string]string{"REFRESH_TOKEN": refreshToken},
	})
	if err != nil {
		return nil, err
	}
	tokens := jwx.RawTokens{
		AccessToken:  *o.AuthenticationResult.AccessToken,
		IdToken:      *o.AuthenticationResult.IdToken,
		RefreshToken: refreshToken,
	}
	_, err = validator.ValidateToken(
		ctx, jwx.EmptyOpaqueClientID, validationURL, tokens.AccessToken)
	if err != nil {
		return nil, err
	}
	_, err = validator.ValidateToken(
		ctx, jwx.EmptyOpaqueClientID, validationURL, tokens.IdToken)
	if err != nil {
		return nil, err
	}
	return &tokens, nil
}
