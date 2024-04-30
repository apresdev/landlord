package authx

import (
	"github.com/apresdev/landlord/internal/session"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
)

func UserPasswordAuth(userPoolName, email, password string) error {
	up, err := GetUserPoolByName(userPoolName)
	if err != nil {
		return err
	}
	client, err := getSingletonUserPoolClientDescription(up)
	if err != nil {
		return err
	}
	args := cognitoidentityprovider.InitiateAuthInput{
		AuthFlow:       "USER_PASSWORD_AUTH",
		ClientId:       client.ClientId,
		AuthParameters: map[string]string{"USERNAME": email, "PASSWORD": password},
	}
	s := session.Must()
	cc := cognitoidentityprovider.NewFromConfig(s.AWS())
	o, err := cc.InitiateAuth(s.Ctx(), &args)
	if err != nil {
		return err
	}
	if _, err = ValidateToken(s, *o.AuthenticationResult.AccessToken, up); err != nil {
		return err
	}
	if _, err = ValidateToken(s, *o.AuthenticationResult.IdToken, up); err != nil {
		return err
	}
	// TODO: validate the refresh token -- cannot use ValidateToken because the
	// refresh token is not a JWT
	s.UserPool = userPoolName
	s.Email = email
	s.AccessToken = *o.AuthenticationResult.AccessToken
	s.IdToken = *o.AuthenticationResult.IdToken
	s.RefreshToken = *o.AuthenticationResult.RefreshToken
	if err = s.SaveToFile(); err != nil {
		return err
	}
	return nil
}
