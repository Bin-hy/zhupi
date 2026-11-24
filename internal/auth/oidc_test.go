package auth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestOIDCFlow(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa: %v", err)
	}

	var issuer string
	var currentNonce string
	mux := http.NewServeMux()

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                issuer,
			"authorization_endpoint":                issuer + "/authorize",
			"token_endpoint":                        issuer + "/token",
			"jwks_uri":                              issuer + "/jwks",
			"response_types_supported":              []string{"code"},
			"subject_types_supported":               []string{"public"},
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]string{{
				"kty": "RSA",
				"kid": "test",
				"use": "sig",
				"alg": "RS256",
				"n":   base64.RawURLEncoding.EncodeToString(key.PublicKey.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(bigEndianBytes(key.PublicKey.E)),
			}},
		})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.Form.Get("code") == "" || r.Form.Get("code_verifier") == "" {
			http.Error(w, "missing code/verifier", http.StatusBadRequest)
			return
		}
		idToken := signIDToken(t, key, issuer, "test-client", currentNonce)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "access",
			"token_type":   "Bearer",
			"expires_in":   3600,
			"id_token":     idToken,
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	issuer = srv.URL

	client := NewOIDCClient(nil)
	cfg := OIDCConfig{
		Issuer:       issuer,
		ClientID:     "test-client",
		ClientSecret: "secret",
		RedirectURL:  "http://app.test/callback",
		Scopes:       []string{"openid", "email", "profile"},
	}
	if err := client.Validate(context.Background(), issuer); err != nil {
		t.Fatalf("validate: %v", err)
	}

	authURL, err := client.AuthCodeURL(context.Background(), cfg, "state-123", "nonce-abc", "verifier-xyz")
	if err != nil {
		t.Fatalf("auth url: %v", err)
	}
	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("parse auth url: %v", err)
	}
	q := parsed.Query()
	if q.Get("state") != "state-123" || q.Get("nonce") != "nonce-abc" {
		t.Fatalf("state/nonce missing: %s", authURL)
	}
	if q.Get("code_challenge") == "" || q.Get("code_challenge_method") != "S256" {
		t.Fatalf("pkce missing: %s", authURL)
	}

	currentNonce = "nonce-abc"
	user, err := client.Exchange(context.Background(), cfg, "code-1", "verifier-xyz", "nonce-abc")
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	if user.Subject != "user-1" || user.Email != "alice@example.com" || !user.EmailVerified {
		t.Fatalf("unexpected user: %+v", user)
	}

	currentNonce = "nonce-wrong"
	if _, err := client.Exchange(context.Background(), cfg, "code-2", "verifier-xyz", "nonce-abc"); err == nil {
		t.Fatal("nonce mismatch should fail")
	}
}

func signIDToken(t *testing.T, key *rsa.PrivateKey, issuer, clientID, nonce string) string {
	t.Helper()
	header := map[string]string{"alg": "RS256", "kid": "test", "typ": "JWT"}
	claims := map[string]any{
		"iss":                issuer,
		"aud":                clientID,
		"sub":                "user-1",
		"exp":                time.Now().Add(time.Hour).Unix(),
		"iat":                time.Now().Unix(),
		"nonce":              nonce,
		"email":              "alice@example.com",
		"email_verified":     true,
		"name":               "Alice",
		"preferred_username": "alice",
		"picture":            "https://example.com/a.png",
	}
	headerJSON, _ := json.Marshal(header)
	claimsJSON, _ := json.Marshal(claims)
	unsigned := base64.RawURLEncoding.EncodeToString(headerJSON) + "." +
		base64.RawURLEncoding.EncodeToString(claimsJSON)
	sum := sha256.Sum256([]byte(unsigned))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func bigEndianBytes(n int) []byte {
	if n == 0 {
		return []byte{0}
	}
	var out []byte
	for n > 0 {
		out = append([]byte{byte(n & 0xff)}, out...)
		n >>= 8
	}
	return out
}
