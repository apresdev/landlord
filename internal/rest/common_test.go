package rest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/apresdev/landlord/internal/appenv"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/apresdev/landlord/internal/db"
	"github.com/apresdev/landlord/internal/jwx"
	"github.com/apresdev/landlord/internal/mocks"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/gin-gonic/gin"
)

type MockApp struct {
	cfg          aws.Config
	cogIdp       mocks.AWSCognitoIdentityProvider
	validator    mocks.TokenValidator
	ddb          mocks.AWSDynamoDB
	settings     appenv.Settings
	rootTenantID uuid.UUID
}

func (m *MockApp) Cfg() aws.Config {
	return m.cfg
}

func (m *MockApp) CogIdp() authx.AWSCognitoIdentityProvider {
	return &m.cogIdp
}

func (m *MockApp) Validator() jwx.TokenValidator {
	return &m.validator
}

func (m *MockApp) DDB() db.AWSDynamoDB {
	return &m.ddb
}

func (m *MockApp) Settings() appenv.Settings {
	return m.settings
}

func (m *MockApp) RootTenantID() uuid.UUID {
	return m.rootTenantID
}

func NewMockApp() *MockApp {
	return &MockApp{
		settings:     appenv.Settings{},
		rootTenantID: uuid.New(),
	}
}

type ginTest struct {
	t *testing.T
	w *httptest.ResponseRecorder
	c *gin.Context
}

func newGinTest(t *testing.T) *ginTest {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return &ginTest{t: t, w: w, c: c}
}

func (gt *ginTest) withRequest(r *http.Request) *ginTest {
	gt.c.Request = r
	gt.c.Request.Header.Set("Content-Type", "application/json")
	return gt
}

func MustJSONMarshal(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func MustJSONUnmarshal(data []byte, v any) {
	if err := json.Unmarshal(data, v); err != nil {
		panic(err)
	}
}
