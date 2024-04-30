package authx

import (
	"errors"
	"fmt"

	"github.com/apresdev/landlord/internal/session"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/golang-jwt/jwt/v5"
)

func ValidateToken(s *session.Session, tok string, up *types.UserPoolType) (*jwt.Token, error) {
	keySet, err := loadJWK(s.AWS().Region, *up.Id)
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
