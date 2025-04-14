package cmd

import (
	"net/url"

	"github.com/apresdev/landlord/internal/appcore"
	"github.com/apresdev/landlord/internal/jwx"
	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/cobra"
)

func ifSet(s, setV, unsetV string) string {
	if s == "" {
		return unsetV
	}
	return setV
}

func displayToken(cmd *cobra.Command, tokName string, tok *jwt.Token) error {
	desc, err := jwx.Describe(tokName, tok)
	if err != nil {
		return err
	}
	cmd.Printf("%s", desc)
	return nil
}

// parseUnverified creates a jwt.Token instance, but does not verify the signature.
// Useful in some diagnostic / troubleshooting scenarios.
func parseUnverified(tokStr string) (*jwt.Token, error) {
	p := jwt.NewParser()
	claims := new(jwt.MapClaims)
	tk, _, err := p.ParseUnverified(tokStr, claims)
	if err != nil {
		return nil, err
	}
	return tk, nil
}

func parseToken(
	cs appcore.Clients,
	validationURL *url.URL,
	tokStr string,
	noValidate bool) (*jwt.Token, error) {

	if noValidate {
		return parseUnverified(tokStr)
	} else {
		return cs.Validator().ValidateToken(cs.Ctx(), jwx.EmptyOpaqueClientID, validationURL, tokStr)
	}
}

func infoCmd() *cobra.Command {
	var raw bool
	var verbose bool
	var noValidate bool
	c := &cobra.Command{
		Use:   "info",
		Short: "Display info about the current session",
		RunE: func(cmd *cobra.Command, args []string) error {
			s := MustSession()
			cmd.Printf("Tokens:\n")
			cs := appcoreMust(cmd.Context())
			if verbose {
				if s.AccessToken == "" {
					cmd.Printf("  Access  : empty\n")
				} else {
					accessJWT, err := parseToken(cs, s.GetValidationURL(), s.AccessToken, noValidate)
					if err != nil {
						return err
					}
					if err := displayToken(cmd, "Access", accessJWT); err != nil {
						return err
					}
				}
				if s.IdToken == "" {
					cmd.Printf("  ID      : empty\n")
				} else {
					idJWT, err := parseToken(cs, s.GetValidationURL(), s.IdToken, noValidate)
					if err != nil {
						return err
					}
					if err = displayToken(cmd, "ID", idJWT); err != nil {
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
	c.PersistentFlags().BoolVarP(
		&verbose, "verbose", "v", false, "verbose output")
	c.PersistentFlags().BoolVarP(
		&raw, "raw", "r", false, "display raw tokens")
	c.PersistentFlags().BoolVarP(
		&noValidate, "no-validate", "N", false, "do not attempt to validate the token")
	return c
}
