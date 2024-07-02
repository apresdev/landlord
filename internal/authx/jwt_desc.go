package authx

import (
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

func JWTDesc(tokName string, tok *jwt.Token) (string, error) {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  %s:\n", tokName))
	sb.WriteString(fmt.Sprintf("    Valid : %v\n", tok.Valid))
	if tok.Method != nil {
		sb.WriteString(fmt.Sprintf("    Method: %s\n", tok.Method.Alg()))
	}
	if len(tok.Header) > 0 {
		sb.WriteString(fmt.Sprintf("    Header:\n"))
		for k, v := range tok.Header {
			sb.WriteString(fmt.Sprintf("      %s: %v\n", k, v))
		}
	} else {
		sb.WriteString(fmt.Sprintf("    Header: empty\n"))
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
			sb.WriteString(fmt.Sprintf("    Claims:\n"))
			for k, v := range claims {
				sb.WriteString(fmt.Sprintf("      %s: %v\n", k, v))
			}
		}
	}
	return sb.String(), nil
}
