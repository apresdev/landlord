package authx

import (
	"github.com/apresdev/landlord/internal/session"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
)

func RefreshAuth(s *session.Session) error {
	up, err := getUserPoolByName(s.UserPool)
	if err != nil {
		return err
	}
	client, err := getSingletonUserPoolClientDescription(up)
	if err != nil {
		return err
	}
	args := cognitoidentityprovider.InitiateAuthInput{
		AuthFlow:       "REFRESH_TOKEN_AUTH",
		ClientId:       client.ClientId,
		AuthParameters: map[string]string{"REFRESH_TOKEN": s.RefreshToken},
	}
	cc := cognitoidentityprovider.NewFromConfig(s.AWS())
	o, err := cc.InitiateAuth(s.Ctx(), &args)
	if err != nil {
		return err
	}
	_, err = ValidateToken(s, s.AccessToken, up)
	if err != nil {
		return err
	}
	_, err = ValidateToken(s, s.IdToken, up)
	if err != nil {
		return err
	}
	s.AccessToken = *o.AuthenticationResult.AccessToken
	s.IdToken = *o.AuthenticationResult.IdToken
	return s.SaveToFile()
}
