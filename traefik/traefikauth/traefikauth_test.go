package traefikauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

// signLikeAuthService mirrors auth-service/jwt.go byte-for-byte so this test
// proves the plugin verifies exactly what the auth-service issues.
func signLikeAuthService(secret string, uid int, exp int64) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload, _ := json.Marshal(map[string]any{"sub": "a@b.c", "uid": uid, "iat": time.Now().Unix(), "exp": exp})
	signingInput := header + "." + base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func TestVerifyJWT(t *testing.T) {
	const secret = "testsecret"

	t.Run("valid token returns uid", func(t *testing.T) {
		tok := signLikeAuthService(secret, 7, time.Now().Add(time.Hour).Unix())
		uid, ok := verifyJWT(secret, tok)
		if !ok || uid != 7 {
			t.Fatalf("want uid=7 ok=true, got uid=%d ok=%v", uid, ok)
		}
	})

	t.Run("expired token rejected", func(t *testing.T) {
		tok := signLikeAuthService(secret, 7, time.Now().Add(-time.Minute).Unix())
		if _, ok := verifyJWT(secret, tok); ok {
			t.Fatal("expired token accepted")
		}
	})

	t.Run("wrong secret rejected", func(t *testing.T) {
		tok := signLikeAuthService(secret, 7, time.Now().Add(time.Hour).Unix())
		if _, ok := verifyJWT("othersecret", tok); ok {
			t.Fatal("token verified under wrong secret")
		}
	})

	t.Run("tampered uid rejected", func(t *testing.T) {
		tok := signLikeAuthService(secret, 7, time.Now().Add(time.Hour).Unix())
		// Flip a byte in the signature segment.
		b := []byte(tok)
		b[len(b)-1] ^= 0x01
		if _, ok := verifyJWT(secret, string(b)); ok {
			t.Fatal("tampered token accepted")
		}
	})

	t.Run("non-positive uid rejected", func(t *testing.T) {
		tok := signLikeAuthService(secret, 0, time.Now().Add(time.Hour).Unix())
		if _, ok := verifyJWT(secret, tok); ok {
			t.Fatal("uid=0 accepted")
		}
	})
}
