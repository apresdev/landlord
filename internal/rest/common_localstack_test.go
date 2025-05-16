package rest

import (
	"context"
	"math/rand"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/stretchr/testify/assert"

	"github.com/apresdev/landlord/internal/appcore"
	"github.com/apresdev/landlord/internal/appenv"
	"github.com/apresdev/landlord/internal/authx"
	"github.com/apresdev/landlord/internal/db"
	"github.com/apresdev/landlord/internal/multi"
	"github.com/google/uuid"
)

func inviteRootUser(s appenv.Settings, username string) error {
	c := appcore.MustWithSettings(s)
	rootTenant, err := db.GetRootTenant(c.Ctx(), c.DBCtx(), c.RootTenantID())
	if err != nil {
		return err
	}
	_, err = authx.SendInvite(
		c.Ctx(),
		c.DBCtx(),
		c.CogIdp(),
		rootTenant,
		username,
		uuid.Nil,
		"admin",
		multi.MFAPolicyNone)
	return err
}

// generateSortOfRandomPassword generates a random password of the specified length.
// It contains at least one uppercase letter, one lowercase letter, one digit,
// and one special character. Not cryptographically secure, but good enough for testing.
func generateSortOfRandomPassword(l int) string {
	const (
		lowerChars   = "abcdefghijklmnopqrstuvwxyz"
		upperChars   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		digitChars   = "0123456789"
		specialChars = "!@#$%^&*()-_=+[]{}|;:,.<>?/"
		allChars     = lowerChars + upperChars + digitChars + specialChars
	)

	randomNumber := rand.Intn(len(allChars))
	password := make([]byte, l)
	password[0] = lowerChars[randomNumber%len(lowerChars)]
	password[1] = upperChars[randomNumber%len(upperChars)]
	password[2] = digitChars[randomNumber%len(digitChars)]
	password[3] = specialChars[randomNumber%len(specialChars)]

	for i := 4; i < l; i++ {
		password[i] = allChars[(i*randomNumber)%len(allChars)]
	}
	return string(password)
}

type invitedUser struct {
	emailAddress string
	accessToken  string
	refreshToken string
}

func inviteUserAndAuthenticate(t *testing.T, app App) *invitedUser {
	ctx := context.Background()
	emailAddress := uuid.New().String() + "@apres.dev"
	rootTenant, err := db.GetRootTenant(ctx, app.DBCtx(), app.RootTenantID())
	assert.NoError(t, err)
	_, err = authx.SendInvite(
		ctx,
		app.DBCtx(),
		app.CogIdp(),
		rootTenant,
		emailAddress,
		uuid.Nil,
		"admin",
		multi.MFAPolicyNone)
	assert.NoError(t, err)
	pwd := generateSortOfRandomPassword(12)
	_, err = app.CogIdp().AdminSetUserPassword(
		ctx,
		&cognitoidentityprovider.AdminSetUserPasswordInput{
			Password:   aws.String(pwd),
			UserPoolId: aws.String(rootTenant.UserPoolID),
			Username:   aws.String(emailAddress),
			Permanent:  true,
		})
	assert.NoError(t, err)

	upc, err := authx.GetUserPoolClient(ctx, app.CogIdp(), rootTenant.UserPoolID)
	assert.NoError(t, err)

	params := make(map[string]string)
	params["USERNAME"] = emailAddress
	params["PASSWORD"] = pwd
	secretHash, err := authx.CreateSecretHash(
		emailAddress, aws.ToString(upc.ClientId), aws.ToString(upc.ClientSecret))
	assert.NoError(t, err)
	params["SECRET_HASH"] = secretHash

	o, err := app.CogIdp().InitiateAuth(
		ctx,
		&cognitoidentityprovider.InitiateAuthInput{
			AuthFlow:       "USER_PASSWORD_AUTH",
			ClientId:       upc.ClientId,
			AuthParameters: params,
		})
	assert.NoError(t, err)

	return &invitedUser{
		emailAddress: emailAddress,
		accessToken:  aws.ToString(o.AuthenticationResult.AccessToken),
		refreshToken: aws.ToString(o.AuthenticationResult.RefreshToken),
	}
}
