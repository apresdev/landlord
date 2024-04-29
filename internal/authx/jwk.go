package authx

import (
	"github.com/apresdev/landlord/internal/session"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

func jwkURL(region, userPoolID string) string {
	return "https://cognito-idp." + region + ".amazonaws.com/" + userPoolID + "/.well-known/jwks.json"
}

// TODO: caching
func loadJWK(region, userPoolID string) (jwk.Set, error) {
	s := session.Must()
	keySet, err := jwk.Fetch(s.Ctx(), jwkURL(region, userPoolID))
	if err != nil {
		return nil, err
	}
	return keySet, nil
}
