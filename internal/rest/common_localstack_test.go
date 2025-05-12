package rest

import (
	"math/rand"

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
	return authx.SendInvite(
		c.Ctx(),
		c.DBCtx(),
		c.CogIdp(),
		rootTenant,
		username,
		uuid.Nil,
		"admin",
		multi.MFAPolicyNone)
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
