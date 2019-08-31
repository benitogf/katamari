package auth

import (
	"errors"
	"time"

	"github.com/benitogf/jwt"
)

// JwtStore :
type JwtStore struct {
	tokenKey    []byte
	expireAfter time.Duration
}

// JwtToken :
type JwtToken struct {
	tokenKey []byte
	jwt.Token
}

// Claims :
func (t *JwtToken) Claims(key string) interface{} {
	claims := t.Token.Claims.(jwt.MapClaims)
	return claims[key]
}

// SetClaim :
func (t *JwtToken) SetClaim(key string, value interface{}) ClaimSetter {
	claims := t.Token.Claims.(jwt.MapClaims)
	claims[key] = value
	return t
}

// Expiry :
func (t *JwtToken) Expiry() time.Time {
	expt := t.Claims("exp")
	return time.Unix(0, int64(expt.(float64)))
}

// IsExpired :
func (t *JwtToken) IsExpired() bool {
	exp := t.Expiry()
	return time.Now().After(exp)
}

// String :
func (t *JwtToken) String() string {
	tokenStr, _ := t.Token.SignedString(t.tokenKey)
	return tokenStr
}

// NewToken :
func (s *JwtStore) NewToken() *JwtToken {
	token := jwt.New(jwt.GetSigningMethod("HS256"))
	claims := token.Claims.(jwt.MapClaims)
	claims["exp"] = time.Now().Add(s.expireAfter).UnixNano()
	t := &JwtToken{
		tokenKey: s.tokenKey,
		Token:    *token,
	}
	return t
}

// CheckToken :
func (s *JwtStore) CheckToken(token string) (Token, error) {
	t, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return s.tokenKey, nil
	})
	if err != nil {
		return nil, err
	}
	jtoken := &JwtToken{s.tokenKey, *t}
	if jtoken.IsExpired() {
		return jtoken, errors.New("Token expired")
	}
	return jtoken, nil
}

// NewJwtStore :
func NewJwtStore(tokenKey string, expireAfter time.Duration) *JwtStore {
	return &JwtStore{
		[]byte(tokenKey),
		expireAfter,
	}
}
