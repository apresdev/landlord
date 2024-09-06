package authx

import (
	"context"
	"fmt"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
)

type UserMFAResult struct {
	OtpAuthURI string
}

func AssociateSoftwareToken(
	ctx context.Context,
	idp AWSCognitoIdentityProvider,
	appName string,
	accessToken string,
	email string) (*UserMFAResult, error) {

	o, err := idp.AssociateSoftwareToken(ctx,
		&cognitoidentityprovider.AssociateSoftwareTokenInput{
			AccessToken: aws.String(accessToken)},
	)
	if err != nil {
		return nil, err
	}
	otpAuthURI := fmt.Sprintf("otpauth://totp/%s:%s?secret=%s",
		url.PathEscape(appName), url.PathEscape(email), url.QueryEscape(*o.SecretCode))

	return &UserMFAResult{
		OtpAuthURI: otpAuthURI,
	}, nil
}

func VerifySoftwareToken(
	ctx context.Context,
	idp AWSCognitoIdentityProvider,
	userPoolID string,
	accessToken string,
	username string,
	userCode string) error {

	_, err := idp.VerifySoftwareToken(ctx,
		&cognitoidentityprovider.VerifySoftwareTokenInput{
			UserCode:    aws.String(userCode),
			AccessToken: aws.String(accessToken)},
		// TODO: friendly device name storage
	)
	if err != nil {
		return fmt.Errorf("could not verify software token for %s in %s code=%s: %w",
			username, userPoolID, userCode, err)
	}
	_, err = idp.AdminSetUserMFAPreference(ctx,
		&cognitoidentityprovider.AdminSetUserMFAPreferenceInput{
			UserPoolId: aws.String(userPoolID),
			Username:   aws.String(username),
			SoftwareTokenMfaSettings: &types.SoftwareTokenMfaSettingsType{
				Enabled: true, PreferredMfa: true},
		})
	if err != nil {
		return fmt.Errorf("could not set MFA preference for %s in %s: %w",
			username, userPoolID, err)
	}
	return err
}
