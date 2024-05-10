package authx

import (
	"encoding/base64"

	"github.com/google/uuid"
)

const MaxPagingResults = 10

// newNextToken generates an uuid4 and returns in base64 encoding. These
// next tokens are used to paginate through values from AWS APIs.
func newNextToken() string {
	return base64.StdEncoding.EncodeToString([]byte(uuid.New().String()))
}
