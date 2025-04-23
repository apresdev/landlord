package cmd

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"

	"github.com/apresdev/landlord/internal/db"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"

	"github.com/google/uuid"

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

// parseUnverified creates a jwt.Token instance but does not verify the signature.
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

func infoSessionCmd() *cobra.Command {
	var raw bool
	var verbose bool
	var noValidate bool
	c := &cobra.Command{
		Use:   "session",
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

func base64DecodeStringAsBinaryUUID(s string) (uuid.UUID, error) {
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to decode base64 string: %w", err)
	}
	if len(decoded) != 16 {
		return uuid.Nil, fmt.Errorf("decoded string is not a valid UUID: %s", s)
	}
	return uuid.FromBytes(decoded)
}

func infoDecodeJWTCmd() *cobra.Command {
	var in string
	c := &cobra.Command{
		Use:   "decode-jwt",
		Short: "Decode a base64 encoded JWT, without verification",
		RunE: func(cmd *cobra.Command, args []string) error {
			j, err := parseUnverified(in)
			if err != nil {
				return err
			}
			cmd.Println(jwx.Describe("JWT", j))
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&in, "input", "i", "", "base64 encoded binary UUID")
	return c
}

func infoDecodeUUIDCmd() *cobra.Command {
	var in string
	c := &cobra.Command{
		Use:   "decode-uuid",
		Short: "Decode a base64 encoded binary UUID",
		RunE: func(cmd *cobra.Command, args []string) error {
			v, err := base64DecodeStringAsBinaryUUID(in)
			if err != nil {
				return err
			}
			cmd.Println(v.String())
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&in, "input", "i", "", "base64 encoded binary UUID")
	return c
}

func infoDriftCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "drift",
		Short: "Detect any drift between Cognito and Landlord",
		RunE: func(cmd *cobra.Command, args []string) error {
			cs := appcoreMust(cmd.Context())
			tenants, err := db.GetAllTenants(cs.Ctx(), cs.DBCtx())
			if err != nil {
				return err
			}
			for _, t := range tenants {
				tenantUsers, err := db.GetUsersForTenant(cs.Ctx(), cs.DBCtx(), t.ID)
				if err != nil {
					return err
				}
				fmt.Printf("> Tenant %s (%s)\n", t.Name, t.ID.String())
				for _, u := range tenantUsers {
					cogU, err := cs.CogIdp().AdminGetUser(cs.Ctx(), &cognitoidentityprovider.AdminGetUserInput{
						UserPoolId: aws.String(t.UserPoolID),
						Username:   aws.String(u.Username),
					})
					if u.ID != u.CognitoID {
						fmt.Printf(">> User %s (%s %s)\n",
							u.Username, u.ID.String(), u.CognitoID.String())
					} else {
						fmt.Printf(">> User %s (%s)\n",
							u.Username, u.ID.String())
					}
					if err != nil {
						var userNotFoundErr *types.UserNotFoundException
						if errors.As(err, &userNotFoundErr) {
							cmd.Printf("User %s not found in Cognito\n", u.Username)
						} else {
							cmd.Printf("Error getting user %s from Cognito: %v\n", u.Username, err)
						}
					} else {
						cogUUID, err := uuid.Parse(aws.ToString(cogU.Username))
						if err != nil {
							fmt.Printf("Failed to parse Cognito UUID: %s", err)
						} else {
							if cogUUID != u.CognitoID {
								fmt.Printf("Cognito UUID %s does not match Landlord UUID %s\n",
									cogUUID.String(), u.CognitoID.String())
							}
						}

						cogGroups, err := cs.CogIdp().AdminListGroupsForUser(
							cs.Ctx(), &cognitoidentityprovider.AdminListGroupsForUserInput{
								UserPoolId: aws.String(t.UserPoolID),
								Username:   aws.String(u.Username),
							})
						if err != nil {
							cmd.Printf("Error getting groups for user %s: %v\n", u.Username, err)
						} else {
							if len(cogGroups.Groups) == 0 {
								cmd.Printf("User %s is not in any groups\n", u.Username)
							} else if len(cogGroups.Groups) > 1 {
								cmd.Printf("User %s is in multiple groups: ", u.Username)
							} else {
								cogGroup := cogGroups.Groups[0]
								userGroupName := t.GroupName(u.Role)
								if userGroupName != aws.ToString(cogGroup.GroupName) {
									cmd.Printf("User %s has role %s, but is in group %s\n",
										u.Username, u.Role, aws.ToString(cogGroup.GroupName))
								}
							}
						}
					}
				}
			}
			return nil
		},
	}
	return c
}

func infoCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "info",
		Short: "Info sub-command",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	c.AddCommand(infoDecodeJWTCmd())
	c.AddCommand(infoDecodeUUIDCmd())
	c.AddCommand(infoDriftCmd())
	c.AddCommand(infoSessionCmd())
	return c
}
