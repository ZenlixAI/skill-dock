package identity

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	ErrTokenInvalid = errors.New("invalid access token")
	ErrTokenExpired = errors.New("access token expired")
)

type VerifierConfig struct {
	Endpoint     string
	Issuer       string
	Audience     string
	JWKSURL      string
	HTTPClient   *http.Client
	Now          func() time.Time
	DiscoveryTTL time.Duration
}

type Verifier struct {
	issuer    string
	audience  string
	jwksURL   string
	client    *http.Client
	now       func() time.Time
	ttl       time.Duration
	mu        sync.RWMutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
}

type discoveryDocument struct {
	Issuer  string `json:"issuer"`
	JWKSURI string `json:"jwks_uri"`
}

type jwksDocument struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	KID string `json:"kid"`
	KTY string `json:"kty"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type rawClaims struct {
	Issuer            string          `json:"iss"`
	Subject           string          `json:"sub"`
	Audience          json.RawMessage `json:"aud"`
	ClientID          string          `json:"client_id"`
	Email             string          `json:"email"`
	EmailVerified     bool            `json:"email_verified"`
	Name              string          `json:"name"`
	Picture           string          `json:"picture"`
	Scope             string          `json:"scope"`
	OrganizationID    string          `json:"organization_id"`
	Organizations     []string        `json:"organizations"`
	OrganizationRoles []string        `json:"organization_roles"`
	Roles             []string        `json:"roles"`
	ExpiresAt         int64           `json:"exp"`
	NotBefore         int64           `json:"nbf"`
	IssuedAt          int64           `json:"iat"`
}

func NewVerifier(ctx context.Context, config VerifierConfig) (*Verifier, error) {
	client := config.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	now := config.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	ttl := config.DiscoveryTTL
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	issuer := strings.TrimRight(strings.TrimSpace(config.Issuer), "/")
	jwksURL := strings.TrimSpace(config.JWKSURL)
	endpoint := strings.TrimRight(strings.TrimSpace(config.Endpoint), "/")

	if endpoint != "" && (issuer == "" || jwksURL == "") {
		discovery, err := fetchDiscovery(ctx, client, endpoint)
		if err != nil {
			return nil, err
		}
		if issuer == "" {
			issuer = strings.TrimRight(discovery.Issuer, "/")
		}
		if jwksURL == "" {
			jwksURL = discovery.JWKSURI
		}
	}
	if issuer == "" {
		return nil, errors.New("logto issuer is required")
	}
	if jwksURL == "" {
		return nil, errors.New("logto JWKS URL is required")
	}
	if strings.TrimSpace(config.Audience) == "" {
		return nil, errors.New("logto API resource audience is required")
	}

	verifier := &Verifier{
		issuer:   issuer,
		audience: strings.TrimSpace(config.Audience),
		jwksURL:  jwksURL,
		client:   client,
		now:      now,
		ttl:      ttl,
		keys:     map[string]*rsa.PublicKey{},
	}
	if err := verifier.refresh(ctx); err != nil {
		return nil, err
	}
	return verifier, nil
}

func (v *Verifier) Verify(ctx context.Context, token string) (Claims, error) {
	header, payload, signingInput, signature, err := splitJWT(token)
	if err != nil {
		return Claims{}, err
	}
	var headerValue struct {
		Alg string `json:"alg"`
		KID string `json:"kid"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(header, &headerValue); err != nil {
		return Claims{}, ErrTokenInvalid
	}
	if headerValue.Alg != "RS256" {
		return Claims{}, fmt.Errorf("%w: unsupported alg", ErrTokenInvalid)
	}
	if headerValue.KID == "" {
		return Claims{}, fmt.Errorf("%w: missing kid", ErrTokenInvalid)
	}

	key, err := v.key(ctx, headerValue.KID)
	if err != nil {
		return Claims{}, err
	}
	digest := sha256.Sum256(signingInput)
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], signature); err != nil {
		return Claims{}, fmt.Errorf("%w: signature", ErrTokenInvalid)
	}

	var raw rawClaims
	if err := json.Unmarshal(payload, &raw); err != nil {
		return Claims{}, ErrTokenInvalid
	}
	now := v.now()
	if raw.ExpiresAt == 0 || now.After(time.Unix(raw.ExpiresAt, 0)) {
		return Claims{}, ErrTokenExpired
	}
	if raw.NotBefore > 0 && now.Add(30*time.Second).Before(time.Unix(raw.NotBefore, 0)) {
		return Claims{}, fmt.Errorf("%w: nbf", ErrTokenInvalid)
	}
	if strings.TrimRight(raw.Issuer, "/") != v.issuer {
		return Claims{}, fmt.Errorf("%w: issuer", ErrTokenInvalid)
	}
	if !audienceContains(raw.Audience, v.audience) {
		return Claims{}, fmt.Errorf("%w: audience", ErrTokenInvalid)
	}
	if strings.TrimSpace(raw.Subject) == "" {
		return Claims{}, fmt.Errorf("%w: subject", ErrTokenInvalid)
	}

	return Claims{
		Issuer:            raw.Issuer,
		Subject:           raw.Subject,
		ClientID:          raw.ClientID,
		Email:             raw.Email,
		EmailVerified:     raw.EmailVerified,
		Name:              raw.Name,
		Picture:           raw.Picture,
		Scope:             raw.Scope,
		OrganizationID:    raw.OrganizationID,
		Organizations:     raw.Organizations,
		OrganizationRoles: raw.OrganizationRoles,
		Roles:             raw.Roles,
		ExpiresAt:         time.Unix(raw.ExpiresAt, 0),
		NotBefore:         time.Unix(raw.NotBefore, 0),
	}, nil
}

func (v *Verifier) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	key := v.keys[kid]
	fetchedAt := v.fetchedAt
	v.mu.RUnlock()
	if key != nil && v.now().Sub(fetchedAt) < v.ttl {
		return key, nil
	}
	if err := v.refresh(ctx); err != nil {
		return nil, err
	}
	v.mu.RLock()
	defer v.mu.RUnlock()
	key = v.keys[kid]
	if key == nil {
		return nil, fmt.Errorf("%w: unknown kid", ErrTokenInvalid)
	}
	return key, nil
}

func (v *Verifier) refresh(ctx context.Context) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return err
	}
	response, err := v.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch logto JWKS: %s", response.Status)
	}
	var document jwksDocument
	if err := json.NewDecoder(response.Body).Decode(&document); err != nil {
		return err
	}
	keys := map[string]*rsa.PublicKey{}
	for _, item := range document.Keys {
		if item.KTY != "RSA" || item.N == "" || item.E == "" {
			continue
		}
		nBytes, err := base64.RawURLEncoding.DecodeString(item.N)
		if err != nil {
			continue
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(item.E)
		if err != nil {
			continue
		}
		publicKey := &rsa.PublicKey{
			N: new(big.Int).SetBytes(nBytes),
			E: int(new(big.Int).SetBytes(eBytes).Int64()),
		}
		keys[item.KID] = publicKey
	}
	if len(keys) == 0 {
		return errors.New("logto JWKS contains no RSA keys")
	}
	v.mu.Lock()
	v.keys = keys
	v.fetchedAt = v.now()
	v.mu.Unlock()
	return nil
}

func fetchDiscovery(ctx context.Context, client *http.Client, endpoint string) (discoveryDocument, error) {
	url := endpoint + "/oidc/.well-known/openid-configuration"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return discoveryDocument{}, err
	}
	response, err := client.Do(request)
	if err != nil {
		return discoveryDocument{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return discoveryDocument{}, fmt.Errorf("fetch logto discovery: %s", response.Status)
	}
	var document discoveryDocument
	if err := json.NewDecoder(response.Body).Decode(&document); err != nil {
		return discoveryDocument{}, err
	}
	if document.Issuer == "" || document.JWKSURI == "" {
		return discoveryDocument{}, errors.New("logto discovery is incomplete")
	}
	return document, nil
}

func splitJWT(token string) (header []byte, payload []byte, signingInput []byte, signature []byte, err error) {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 3 {
		return nil, nil, nil, nil, ErrTokenInvalid
	}
	header, err = base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, nil, nil, nil, ErrTokenInvalid
	}
	payload, err = base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, nil, nil, nil, ErrTokenInvalid
	}
	signature, err = base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, nil, nil, nil, ErrTokenInvalid
	}
	return header, payload, []byte(parts[0] + "." + parts[1]), signature, nil
}

func audienceContains(raw json.RawMessage, expected string) bool {
	if len(raw) == 0 {
		return false
	}
	var one string
	if err := json.Unmarshal(raw, &one); err == nil {
		return one == expected
	}
	var many []string
	if err := json.Unmarshal(raw, &many); err != nil {
		return false
	}
	for _, audience := range many {
		if audience == expected {
			return true
		}
	}
	return false
}
