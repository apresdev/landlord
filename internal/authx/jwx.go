package authx

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"go.uber.org/zap"

	"github.com/apresdev/landlord/internal/localstack"

	"github.com/golang-jwt/jwt/v5"

	"github.com/apresdev/landlord/internal/appenv"

	"github.com/lestrrat-go/jwx/v2/jwk"
)

func ValidationURL(region, userPoolID string) *url.URL {
	raw := awsValidationURL(region, userPoolID)
	if appenv.LocalStack {
		raw = localstackValidationURL(userPoolID)
	}
	u, err := url.Parse(raw)
	if err != nil {
		zap.L().Fatal("could not parse URL", zap.Error(err))
	}
	return u
}

func awsValidationURL(region, userPoolID string) string {
	return "https://cognito-idp." + region + ".amazonaws.com/" + userPoolID + "/.well-known/jwks.json"
}

func localstackValidationURL(userPoolID string) string {
	u, err := url.JoinPath(localstack.Endpoint(), userPoolID, ".well-known/jwks.json")
	if err != nil {
		zap.L().Fatal("could not join URL path", zap.Error(err))
	}
	return u
}

var (
	cachedJWKSet     jwk.Set
	clientRequesters = make(map[string]int)
)

func loadJWK(ctx context.Context, validationURL *url.URL) (jwk.Set, error) {
	if cachedJWKSet != nil {
		return cachedJWKSet, nil
	}
	if appenv.LocalStack {
		keySet, err := jwk.Fetch(ctx, validationURL.String())
		if err != nil {
			return nil, err
		}
		cachedJWKSet = keySet
	} else {
		keySet, err := jwk.Fetch(ctx, validationURL.String())
		if err != nil {
			return nil, err
		}
		cachedJWKSet = keySet
	}
	return cachedJWKSet, nil
}

func reloadJWK(ctx context.Context, validationURL *url.URL) (jwk.Set, error) {
	cachedJWKSet = nil
	return loadJWK(ctx, validationURL)
}

type OpaqueClientID string

var EmptyOpaqueClientID = OpaqueClientID("")

//go:generate mockery --output ../mocks --name TokenValidator --with-expecter=true
type TokenValidator interface {
	ValidateToken(
		ctx context.Context,
		cid OpaqueClientID,
		validationURL *url.URL,
		tok string) (*jwt.Token, error)
}

func strFromMap(m map[string]interface{}, key string) (string, error) {
	v, ok := m[key]
	if !ok {
		return "", fmt.Errorf("key %s not found", key)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("value for key %s is not a string", key)
	}
	return s, nil
}

type DefaultTokenValidator struct{}

func (DefaultTokenValidator) ValidateToken(
	ctx context.Context,
	cid OpaqueClientID,
	validationURL *url.URL,
	tok string) (*jwt.Token, error) {

	if tok == "" {
		return nil, errors.New("token is empty")
	}
	keySet, err := loadJWK(ctx, validationURL)
	if err != nil {
		return nil, err
	}
	token, err := jwt.Parse(tok, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		kid, err := strFromMap(token.Header, "kid")
		if err != nil {
			return nil, errors.New("kid header not found")
		}
		keys, ok := keySet.LookupKeyID(kid)
		if !ok {
			if cid == "" {
				zap.L().Warn("cannot reload JWK set without client ID", zap.String("kid", kid))
				return nil, errors.New("key with specified kid is not present in JWK set (1)")
			}
			clientRequesters[string(cid)]++
			if clientRequesters[string(cid)] > 1 {
				zap.L().Warn("cannot reload JWK, client ID has already resulted in a reset",
					zap.String("kid", kid))
				return nil, errors.New("key with specified kid is not present in JWK set (2)")
			}
			keySet, err = reloadJWK(ctx, validationURL)
			if err != nil {
				return nil, err
			}
			keys, ok = keySet.LookupKeyID(kid)
			if !ok {
				return nil, errors.New("key with specified kid is not present in JWK set (3)")
			}
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

type Tokens struct {
	AccessToken *jwt.Token
	IdToken     *jwt.Token
	// The RefreshToken is base64-encoded data. It is not a JWT token.
	RefreshToken string
}

func ValidateAllTokens(
	ctx context.Context,
	cid OpaqueClientID,
	v TokenValidator,
	validationURL *url.URL,
	rawTokens RawTokens) (*Tokens, error) {

	result := Tokens{}
	var err error
	result.AccessToken, err = v.ValidateToken(
		ctx, cid, validationURL, rawTokens.AccessToken)
	if err != nil {
		return nil, err
	}
	result.IdToken, err = v.ValidateToken(
		ctx, cid, validationURL, rawTokens.IdToken)
	if err != nil {
		return nil, err
	}
	result.RefreshToken = rawTokens.RefreshToken
	return &result, nil
}
