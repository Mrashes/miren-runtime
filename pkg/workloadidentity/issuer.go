package workloadidentity

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"miren.dev/runtime/pkg/cloudauth"
)

type IssuerConfig struct {
	DataPath       string
	IssuerURL      string
	OrganizationID string
	ClusterID      string
}

type Issuer struct {
	privateKey     ed25519.PrivateKey
	publicKey      ed25519.PublicKey
	kid            string
	issuerURL      string
	organizationID string
	clusterID      string
}

type WorkloadClaims struct {
	jwt.RegisteredClaims
	OrganizationID string `json:"organization_id,omitempty"`
	ClusterID      string `json:"cluster_id,omitempty"`
	App            string `json:"app,omitempty"`
	SandboxID      string `json:"sandbox_id"`
}

func NewIssuer(cfg IssuerConfig) (*Issuer, error) {
	keyPath := filepath.Join(cfg.DataPath, "server", "workload-identity.key")

	kp, err := loadOrGenerateKey(keyPath)
	if err != nil {
		return nil, fmt.Errorf("workload identity key: %w", err)
	}

	kid := computeKID(kp.PublicKey)

	return &Issuer{
		privateKey:     kp.PrivateKey,
		publicKey:      kp.PublicKey,
		kid:            kid,
		issuerURL:      cfg.IssuerURL,
		organizationID: cfg.OrganizationID,
		clusterID:      cfg.ClusterID,
	}, nil
}

func (iss *Issuer) IssuerURL() string {
	return iss.issuerURL
}

func (iss *Issuer) PublicKey() any {
	return iss.publicKey
}

func (iss *Issuer) Hostname() string {
	u, err := url.Parse(iss.issuerURL)
	if err != nil {
		return ""
	}
	return u.Host
}

const (
	DefaultTTL = 1 * time.Hour
	MaxTTL     = 24 * time.Hour
	MinTTL     = 60 * time.Second
)

type TokenOptions struct {
	Audience []string
	TTL      time.Duration
}

func (iss *Issuer) IssueToken(app, sandboxID string) (string, error) {
	return iss.IssueTokenWithOptions(app, sandboxID, TokenOptions{})
}

func (iss *Issuer) IssueTokenWithOptions(app, sandboxID string, opts TokenOptions) (string, error) {
	now := time.Now()

	aud := opts.Audience
	if len(aud) == 0 {
		aud = []string{"miren"}
	}

	ttl := opts.TTL
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	if ttl > MaxTTL {
		ttl = MaxTTL
	}
	if ttl < MinTTL {
		ttl = MinTTL
	}

	sub := buildSubject(iss.organizationID, app, sandboxID)

	claims := WorkloadClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    iss.issuerURL,
			Subject:   sub,
			Audience:  jwt.ClaimStrings(aud),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			ID:        uuid.New().String(),
		},
		OrganizationID: iss.organizationID,
		ClusterID:      iss.clusterID,
		App:            app,
		SandboxID:      sandboxID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	token.Header["kid"] = iss.kid

	return token.SignedString(iss.privateKey)
}

func (iss *Issuer) DiscoveryDocument() []byte {
	doc := map[string]any{
		"issuer":                                iss.issuerURL,
		"jwks_uri":                              iss.issuerURL + "/.well-known/miren/jwks",
		"response_types_supported":              []string{"id_token"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"EdDSA"},
	}

	data, _ := json.Marshal(doc)
	return data
}

func (iss *Issuer) JWKSDocument() ([]byte, error) {
	jwk := jose.JSONWebKey{
		Key:       iss.publicKey,
		KeyID:     iss.kid,
		Algorithm: "EdDSA",
		Use:       "sig",
	}

	jwks := jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{jwk},
	}

	return json.Marshal(jwks)
}

func loadOrGenerateKey(keyPath string) (*cloudauth.KeyPair, error) {
	data, err := os.ReadFile(keyPath)
	if err == nil {
		return cloudauth.LoadKeyPairFromPEM(string(data))
	}

	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("reading key file: %w", err)
	}

	kp, err := cloudauth.GenerateKeyPair()
	if err != nil {
		return nil, err
	}

	pemData, err := kp.PrivateKeyPEM()
	if err != nil {
		return nil, fmt.Errorf("encoding private key: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(keyPath), 0700); err != nil {
		return nil, fmt.Errorf("creating key directory: %w", err)
	}

	if err := os.WriteFile(keyPath, []byte(pemData), 0600); err != nil {
		return nil, fmt.Errorf("writing key file: %w", err)
	}

	return kp, nil
}

func computeKID(pub ed25519.PublicKey) string {
	hash := sha256.Sum256(pub)
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

func buildSubject(orgID, app, sandboxID string) string {
	var parts []string
	if orgID != "" {
		parts = append(parts, "org:"+orgID)
	}
	if app != "" {
		parts = append(parts, "app:"+app)
	}
	parts = append(parts, "sandbox:"+sandboxID)
	return strings.Join(parts, ":")
}
