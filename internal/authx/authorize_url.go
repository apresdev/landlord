package authx

import (
	"context"
	"net/url"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
)

func CreateAuthorizeURL(
	ctx context.Context,
	idp AWSCognitoIdentityProvider,
	userPoolName string,
	redirectURI string) (string, error) {

	up, err := GetUserPoolByNameOrID(ctx, idp, userPoolName)
	if err != nil {
		return "", err
	}
	upc, err := GetSingletonUserPoolClient(ctx, idp, up)
	if err != nil {
		return "", err
	}
	base := os.Getenv("LANDLORD_CUSTOM_DOMAIN")
	u, err := url.JoinPath(base, "/oauth2/authorize")
	if err != nil {
		return "", err
	}
	query := url.Values{}
	query.Add("client_id", aws.ToString(upc.ClientId))
	query.Add("response_type", "code")
	query.Add("scope", "aws.cognito.signin.user.admin email openid phone")
	query.Add("redirect_uri", redirectURI)
	u = u + "?" + query.Encode()
	return u, nil
}
