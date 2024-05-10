package cmd

import (
	"github.com/apresdev/landlord/internal/authx"
	"github.com/golang-jwt/jwt/v5"

	"github.com/spf13/cobra"
)

func ifSet(s, setV, unsetV string) string {
	if s == "" {
		return unsetV
	}
	return setV
}

func displayToken(cmd *cobra.Command, tokName string, tok jwt.Token) error {
	cmd.Printf("  %s:\n", tokName)
	cmd.Printf("    Valid : %v\n", tok.Valid)
	if tok.Method != nil {
		cmd.Printf("    Method: %s\n", tok.Method.Alg())
	}
	if len(tok.Header) > 0 {
		cmd.Printf("    Header:\n")
		for k, v := range tok.Header {
			cmd.Printf("      %s: %v\n", k, v)
		}
	} else {
		cmd.Printf("    Header: empty\n")
	}
	if tok.Claims != nil {
		issuer, err := tok.Claims.GetIssuer()
		if err != nil {
			return err
		}
		subject, err := tok.Claims.GetSubject()
		if err != nil {
			return err
		}
		cmd.Printf("    Issuer: %s\n", issuer)
		cmd.Printf("    Subject: %s\n", subject)
		if claims, ok := tok.Claims.(jwt.MapClaims); ok {
			cmd.Printf("    Claims:\n")
			for k, v := range claims {
				cmd.Printf("      %s: %v\n", k, v)
			}
		}
	}
	return nil
}

func createInfoCmd() *cobra.Command {
	var raw bool
	var verbose bool
	infoCmd := &cobra.Command{
		Use:   "info",
		Short: "Display info about the current session",
		RunE: func(cmd *cobra.Command, args []string) error {
			s := MustSession()
			cmd.Printf("User pool : %s\n", s.UserPool)
			cmd.Printf("Email     : %s\n", s.Email)
			cmd.Printf("Tokens:\n")
			if verbose {
				up, err := authx.GetUserPoolByNameOrID(s.Ctx, s.CogIdp, s.UserPool)
				if err != nil {
					return err
				}
				if s.AccessToken == "" {
					cmd.Printf("  Access  : empty\n")
				} else {
					accessJWT, err := s.Validator.ValidateToken(s.Ctx, s.GetRegion(), up, s.AccessToken)
					if err != nil {
						return err
					}
					if err := displayToken(cmd, "Access", *accessJWT); err != nil {
						return err
					}
				}
				if s.IdToken == "" {
					cmd.Printf("  ID      : empty\n")
				} else {
					idJWT, err := s.Validator.ValidateToken(s.Ctx, s.GetRegion(), up, s.IdToken)
					if err != nil {
						return err
					}
					if err = displayToken(cmd, "ID", *idJWT); err != nil {
						return err
					}
				}
			} else {
				cmd.Printf("  Access  : %s\n", ifSet(s.AccessToken, "valid", "empty"))
				cmd.Printf("  ID      : %s\n", ifSet(s.IdToken, "valid", "empty"))
				cmd.Printf("  Refresh : %s\n", ifSet(s.RefreshToken, "valid", "empty"))
			}
			if raw {
				cmd.Printf("Access:\n%s\n", s.AccessToken)
				cmd.Printf("ID:\n%s\n", s.IdToken)
				cmd.Printf("Refresh:\n%s\n", s.RefreshToken)
			}
			return nil
		},
	}
	infoCmd.PersistentFlags().BoolVarP(
		&verbose, "verbose", "v", false, "verbose output")
	infoCmd.PersistentFlags().BoolVarP(
		&raw, "raw", "r", false, "display raw tokens")
	return infoCmd
}
