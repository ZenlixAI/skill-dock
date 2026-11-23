package identity

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestVerifierAcceptsSignedAccessToken(t *testing.T) {
	privateKey, server := newOIDCTestServer(t)
	defer server.Close()

	verifier, err := NewVerifier(context.Background(), VerifierConfig{
		Endpoint: server.URL,
		Audience: "https://catalog.example.com/api",
	})
	if err != nil {
		t.Fatalf("new verifier: %v", err)
	}

	token := signTestJWT(t, privateKey, map[string]any{
		"iss":             server.URL + "/oidc",
		"aud":             "https://catalog.example.com/api",
		"sub":             "user_123",
		"client_id":       "web",
		"scope":           "openid catalog:read",
		"exp":             time.Now().Add(time.Hour).Unix(),
		"iat":             time.Now().Unix(),
		"organization_id": "org_logto",
	})
	claims, err := verifier.Verify(context.Background(), token)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if claims.Subject != "user_123" || claims.OrganizationID != "org_logto" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestVerifierRejectsWrongAudienceAndExpiredToken(t *testing.T) {
	privateKey, server := newOIDCTestServer(t)
	defer server.Close()
	verifier, err := NewVerifier(context.Background(), VerifierConfig{
		Endpoint: server.URL,
		Audience: "https://catalog.example.com/api",
	})
	if err != nil {
		t.Fatalf("new verifier: %v", err)
	}

	wrongAudience := signTestJWT(t, privateKey, map[string]any{
		"iss": server.URL + "/oidc",
		"aud": "other-api",
		"sub": "user_123",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	if _, err := verifier.Verify(context.Background(), wrongAudience); err == nil {
		t.Fatal("expected wrong audience to fail")
	}

	expired := signTestJWT(t, privateKey, map[string]any{
		"iss": server.URL + "/oidc",
		"aud": "https://catalog.example.com/api",
		"sub": "user_123",
		"exp": time.Now().Add(-time.Minute).Unix(),
	})
	if _, err := verifier.Verify(context.Background(), expired); err == nil {
		t.Fatal("expected expired token to fail")
	}
}

func newOIDCTestServer(t *testing.T) (*rsa.PrivateKey, *httptest.Server) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("content-type", "application/json")
		switch request.URL.Path {
		case "/oidc/.well-known/openid-configuration":
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"issuer":   server.URL + "/oidc",
				"jwks_uri": server.URL + "/oidc/jwks",
			})
		case "/oidc/jwks":
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"keys": []map[string]any{{
					"kty": "RSA",
					"kid": "test-key",
					"alg": "RS256",
					"n":   base64.RawURLEncoding.EncodeToString(privateKey.N.Bytes()),
					"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.E)).Bytes()),
				}},
			})
		default:
			http.NotFound(writer, request)
		}
	}))
	return privateKey, server
}

func signTestJWT(t *testing.T, privateKey *rsa.PrivateKey, claims map[string]any) string {
	t.Helper()
	header := map[string]any{"alg": "RS256", "typ": "JWT", "kid": "test-key"}
	headerJSON, _ := json.Marshal(header)
	payloadJSON, _ := json.Marshal(claims)
	signingInput := base64.RawURLEncoding.EncodeToString(headerJSON) + "." + base64.RawURLEncoding.EncodeToString(payloadJSON)
	digest := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature)
}
