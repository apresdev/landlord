package authx

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/apresdev/landlord/internal/localstack"

	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/golang-jwt/jwt/v5"

	"github.com/apresdev/landlord/internal/appenv"

	"github.com/lestrrat-go/jwx/v2/jwk"
)

func awsJWKURL(region, userPoolID string) string {
	return "https://cognito-idp." + region + ".amazonaws.com/" + userPoolID + "/.well-known/jwks.json"
}

func localstackJWKURL(userPoolID string) (string, error) {
	return url.JoinPath(localstack.Endpoint(), userPoolID, ".well-known/jwks.json")
}

// TODO: caching
func loadJWK(ctx context.Context, region, userPoolID string) (jwk.Set, error) {
	if appenv.LocalStack {
		u, err := localstackJWKURL(userPoolID)
		if err != nil {
			return nil, err
		}
		keySet, err := jwk.Fetch(ctx, u)
		if err != nil {
			return nil, err
		}
		return keySet, nil
	} else {
		keySet, err := jwk.Fetch(ctx, awsJWKURL(region, userPoolID))
		if err != nil {
			return nil, err
		}
		return keySet, nil
	}
}

//go:generate mockery --output ../mocks --name TokenValidator --with-expecter=true
type TokenValidator interface {
	ValidateToken(
		ctx context.Context,
		region string,
		up *types.UserPoolType,
		tok string) (*jwt.Token, error)
}

type DefaultTokenValidator struct{}

func (DefaultTokenValidator) ValidateToken(
	ctx context.Context,
	region string,
	up *types.UserPoolType,
	tok string) (*jwt.Token, error) {

	keySet, err := loadJWK(ctx, region, *up.Id)
	if err != nil {
		return nil, err
	}
	token, err := jwt.Parse(tok, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, errors.New("kid header not found")
		}
		keys, ok := keySet.LookupKeyID(kid)
		if !ok {
			return nil, errors.New("key with specified kid is not present in jwks")
		}
		var publicKey any
		err = keys.Raw(&publicKey)
		if err != nil {
			return nil, errors.New("could not parse pubkey")
		}
		return publicKey, nil
	})
	if err != nil {
		return nil, err
	}
	return token, nil
}

func NewTokenValidator() TokenValidator {
	return DefaultTokenValidator{}
}
