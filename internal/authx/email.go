package authx

import (
	"context"
	"errors"
	"net/mail"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	sestypes "github.com/aws/aws-sdk-go-v2/service/ses/types"
	"go.uber.org/zap"
)

type Email struct {
	src    string
	domain string
	addr   *mail.Address
}

func ParseEmail(s string) (Email, error) {
	ea, err := mail.ParseAddress(s)
	if err != nil {
		return Email{}, err
	}
	d := ea.Address[strings.Index(ea.Address, "@")+1:]
	if d == "" {
		return Email{}, errors.New("invalid email")
	}
	return Email{src: s, domain: d, addr: ea}, nil
}

func (e Email) Address() string {
	return e.addr.Address
}

func (e Email) Domain() string {
	return e.domain
}

func (e Email) Full() string {
	return e.addr.String()
}

func validEmailDomain(s string) (string, error) {
	if strings.HasPrefix(s, "@") {
		s = s[1:]
	}
	if strings.Contains(s, "@") {
		return "", errors.New("invalid domain")
	}
	if len(s) < 3 {
		return "", errors.New("domain too short")
	}
	return s, nil
}

func newMsg(subject, body string) (*sestypes.Message, error) {
	bodyContent := sestypes.Content{Data: aws.String(body)}
	subjectContent := sestypes.Content{Data: aws.String(subject)}
	return &sestypes.Message{
		Body:    &sestypes.Body{Text: &bodyContent},
		Subject: &subjectContent,
	}, nil
}

func SendAdminEmail(
	ctx context.Context,
	s AWSSES,
	e Email,
	subject string,
	body string) error {

	msg, err := newMsg(subject, body)
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
	zap.L().Info("Sent email",
		zap.String("email", e.Address()), zap.Any("output", o))
	return nil
}
