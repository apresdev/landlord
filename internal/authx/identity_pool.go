package authx

import (
	"fmt"

	"github.com/apresdev/landlord/internal/session"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentity"
)

// TODO: add identity-pool sub-command
func listIdentityPools() error {
	s := session.Must()
	c := cognitoidentity.NewFromConfig(s.AWS())
	maxResults := int32(10)
	nextToken := newNextToken()
	args := cognitoidentity.ListIdentityPoolsInput{
		MaxResults: &maxResults,
		NextToken:  &nextToken,
	}
	o, err := c.ListIdentityPools(s.Ctx(), &args)
	if err != nil {
		return err
	}
	for _, id := range o.IdentityPools {
		fmt.Printf("Identity Pool: %s\n", *id.IdentityPoolName)
	}
	return nil
}
