package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"time"
)

// Claims is the minimal JWT payload. `uid` is the fin-track users.id that the
// Traefik plugin injects downstream as the X-User-ID header.
//
// The token is a hand-rolled HS256 JWT (stdlib only) so that the Traefik local
// plugin — which runs under the Yaegi interpreter and is happiest without third
// party crypto imports — can verify it with the identical algorithm.
type Claims struct {
	Sub string `json:"sub"` // subject (the user's email)
	UID int    `json:"uid"` // fin-track app user id -> X-User-ID
	Iat int64  `json:"iat"`
	Exp int64  `json:"exp"`
}

var errInvalidToken = errors.New("invalid token")

func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func loadFromFile(path string) ([]byte, error) {
	f, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func sign(secret []byte, signingInput string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(signingInput))
	return b64(mac.Sum(nil))
}

// signJWT returns a signed HS256 token for the given claims.
func signJWT(secret []byte, c Claims) (string, error) {
	header := b64([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	signingInput := header + "." + b64(payload)
	return signingInput + "." + sign(secret, signingInput), nil
}

// verifyJWT checks the signature and expiry and returns the claims.
func verifyJWT(secret []byte, token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errInvalidToken
	}
	signingInput := parts[0] + "." + parts[1]
	expected := sign(secret, signingInput)
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return nil, errInvalidToken
	}

	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, errInvalidToken
	}
	var c Claims
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, errInvalidToken
	}
	if c.Exp != 0 && time.Now().Unix() >= c.Exp {
		return nil, errInvalidToken
	}
	return &c, nil
}
