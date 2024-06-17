package authx

import (
	"errors"
	"net/mail"
	"strings"
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
