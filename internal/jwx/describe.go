package jwx

import (
	"fmt"
	"sort"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// Describe returns a string representation of the token suitable for
// display in a command-line or other multi-line text display.
func Describe(tokName string, tok *jwt.Token) (string, error) {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  %s:\n", tokName))
	sb.WriteString(fmt.Sprintf("    Valid : %v\n", tok.Valid))
	if tok.Method != nil {
		sb.WriteString(fmt.Sprintf("    Method: %s\n", tok.Method.Alg()))
	}
	if len(tok.Header) > 0 {
		sb.WriteString("    Header:\n")
		for k, v := range tok.Header {
			sb.WriteString(fmt.Sprintf("      %s: %v\n", k, v))
		}
	} else {
		sb.WriteString("    Header: empty\n")
	}
	if tok.Claims != nil {
		issuer, err := tok.Claims.GetIssuer()
		if err != nil {
			return "", err
		}
		subject, err := tok.Claims.GetSubject()
		if err != nil {
			return "", err
		}
		sb.WriteString(fmt.Sprintf("    Issuer: %s\n", issuer))
		sb.WriteString(fmt.Sprintf("    Subject: %s\n", subject))

		otherClaims, err := mapClaims(tok)
		if err != nil {
			sb.WriteString("    Claims: error not a map\n")
		} else {
			sb.WriteString("    Claims:\n")
			keys := make([]string, 0, len(otherClaims))
			for k := range otherClaims {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				sb.WriteString(fmt.Sprintf("      %s: %v\n", k, otherClaims[k]))
			}
		}
	} else {
		sb.WriteString("    Claims: empty\n")
	}
	return sb.String(), nil
}

// mapClaims returns the claims in the token as a MapClaims. Parsing an
// unverified token results in the data structure being a *jwt.MapClaims.
// The normal code path results in the data structure being a jwt.MapClaims.
func mapClaims(tok *jwt.Token) (jwt.MapClaims, error) {
	if cl, ok := tok.Claims.(jwt.MapClaims); ok {
		return cl, nil
	}
	if cl, ok := tok.Claims.(*jwt.MapClaims); ok {
		return *cl, nil
	}
	return nil, fmt.Errorf("claims are not of type MapClaims")
}
