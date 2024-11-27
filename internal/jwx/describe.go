package jwx

import (
	"fmt"
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
		if claims, ok := tok.Claims.(jwt.MapClaims); ok {
			sb.WriteString("    Claims:\n")
			for k, v := range claims {
				sb.WriteString(fmt.Sprintf("      %s: %v\n", k, v))
			}
		}
	}
	return sb.String(), nil
}
