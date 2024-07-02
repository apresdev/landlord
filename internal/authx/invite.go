package authx

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ses"
	"go.uber.org/zap"

	"github.com/aws/aws-sdk-go-v2/aws"
	sestypes "github.com/aws/aws-sdk-go-v2/service/ses/types"
)

func newInviteMessage(
	ctx context.Context,
	ten *Tenant,
	cogIdp AWSCognitoIdentityProvider) (*sestypes.Message, error) {

	url, err := CreateAuthorizeURL(ctx, cogIdp, ten.UserPoolName, RedirectURI())
	if err != nil {
		zap.L().Error("Error creating authorize URL", zap.Error(err))
		return nil, err
	}
	now := time.Now().String()
	bodyContent := sestypes.Content{
		Data: aws.String("test message body " + now + "\n" + url + "\n"),
	}
	subjectContent := sestypes.Content{
		Data: aws.String("test message subject " + now),
	}
	return &sestypes.Message{
		Body:    &sestypes.Body{Text: &bodyContent},
		Subject: &subjectContent,
	}, nil
}

func SendInviteEmail(
	ctx context.Context,
	s AWSSES,
	cogIdp AWSCognitoIdentityProvider,
	ten *Tenant,
	e Email) error {

	msg, err := newInviteMessage(ctx, ten, cogIdp)
	if err != nil {
		return err
	}
	senderEmail := os.Getenv("LANDLORD_SENDER_EMAIL")
	o, err := s.SendEmail(
		ctx, &ses.SendEmailInput{
			Destination: &sestypes.Destination{ToAddresses: []string{e.Address()}},
			Message:     msg,
			Source:      aws.String(senderEmail),
		})
	if err != nil {
		return err
	}
	zap.L().Info("Sent invite email",
		zap.String("email", e.Address()), zap.Any("output", o))
	return nil
}

func SendInvite(
	ctx context.Context,
	s AWSSES,
	cogIdp AWSCognitoIdentityProvider,
	tens *Tenants,
	email string,
	role string) error {

	e, err := ParseEmail(email)
	if err != nil {
		return errors.New("invalid email")
	}
	ten, err := tens.ForExclusiveEmailDomain(e.Domain())
	if err != nil {
		return err
	}
	if ten.HasUser(e.Address()) {
		return errors.New("user already invited to tenant")
	}
	if err := SendInviteEmail(ctx, s, cogIdp, ten, e); err != nil {
		return err
	}
	return tens.UpdateTenant(ten.ID, func(t *Tenant) error {
		ten.Users = append(ten.Users, User{
			Email:  email,
			Role:   role,
			Status: "invited",
		})
		return nil
	})
}

func ResendInvite(
	ctx context.Context,
	s AWSSES,
	cogIdp AWSCognitoIdentityProvider,
	tens *Tenants,
	email string,
	role string) error {

	e, err := ParseEmail(email)
	if err != nil {
		return errors.New("invalid email")
	}
	ten, err := tens.ForExclusiveEmailDomain(e.Domain())
	if err != nil {
		return err
	}
	if _, err = ten.User(e.Address()); err != nil {
		return errors.New("user not in tenant, cannot resend invite")
	}
	if err := SendInviteEmail(ctx, s, cogIdp, ten, e); err != nil {
		return err
	}
	return tens.UpdateTenant(ten.ID, func(t *Tenant) error {
		return t.UpdateUser(e.Address(), func(u *User) error {
			u.Status = "invited"
			u.Role = role
			return nil
		})
	})
}
