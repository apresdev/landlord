package authx

import (
	"encoding/base64"

	"github.com/google/uuid"
)

const (
	MaxPagingResults = int32(10)
	MaxTotalResults  = 500
)

// NewNextToken generates an uuid4 and returns in base64 encoding. These
// next tokens are used to paginate through values from AWS APIs.
func NewNextToken() string {
	return base64.StdEncoding.EncodeToString([]byte(uuid.New().String()))
}
