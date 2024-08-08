package authx

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/url"

	"github.com/apresdev/landlord/internal/jwx"

	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
)

// CreateSecretHash creates the secret hash for AWS Cognito
func CreateSecretHash(email, clientId, clientSecret string) (string, error) {
	message := email + clientId
	key := []byte(clientSecret)

	h := hmac.New(sha256.New, key)
	_, err := h.Write([]byte(message))
	if err != nil {
		return "", err
	}
	secretHash := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return secretHash, nil
}

func UserPasswordAuth(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	validator jwx.TokenValidator,
	validationURL *url.URL,
	up *types.UserPoolType,
	username string,
	password string) (*jwx.Tokens, error) {

	if validationURL == nil {
		return nil, errors.New("validationURL is nil")
	}
	if up == nil {
		return nil, errors.New("user pool is nil")
	}
	upc, err := GetUserPoolClient(ctx, cogIdp, aws.ToString(up.Id))
	if err != nil {
		return nil, err
	}
	params := make(map[string]string)
	params["USERNAME"] = username
	params["PASSWORD"] = password
	if upc.ClientSecret != nil {
		secretHash, err := CreateSecretHash(
			username, aws.ToString(upc.ClientId), aws.ToString(upc.ClientSecret))
		if err != nil {
			return nil, err
		}
		params["SECRET_HASH"] = secretHash
	}
	o, err := cogIdp.InitiateAuth(ctx,
		&cognitoidentityprovider.InitiateAuthInput{
			AuthFlow:       "USER_PASSWORD_AUTH",
			ClientId:       upc.ClientId,
			AuthParameters: params,
		})
	if err != nil {
		return nil, err
	}
	accessTok, err := validator.ValidateToken(
		ctx,
		jwx.EmptyOpaqueClientID,
		validationURL,
		aws.ToString(o.AuthenticationResult.AccessToken))
	if err != nil {
		return nil, err
	}
	idTok, err := validator.ValidateToken(
		ctx,
		jwx.EmptyOpaqueClientID,
		validationURL,
		aws.ToString(o.AuthenticationResult.IdToken))
	if err != nil {
		return nil, err
	}
	return &jwx.Tokens{
		AccessToken:  accessTok,
		IdToken:      idTok,
		RefreshToken: aws.ToString(o.AuthenticationResult.RefreshToken),
	}, nil
}
