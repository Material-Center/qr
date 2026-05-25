package request

import (
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const TokenTypeOpenAPI = "openapi"

// CustomClaims structure
type CustomClaims struct {
	BaseClaims
	BufferTime int64
	TokenType  string `json:"tokenType,omitempty"`
	jwt.RegisteredClaims
}

type BaseClaims struct {
	UUID        uuid.UUID
	ID          uint
	Username    string
	NickName    string
	AuthorityId uint
}
