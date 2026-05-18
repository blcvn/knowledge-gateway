package usecase_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/vnp-community/vnp-memory/gateway/domain"
	"github.com/vnp-community/vnp-memory/gateway/usecase"
)

var testLogger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

// testKeyPair generates an RSA key pair for testing.
func testKeyPair(t *testing.T) (*rsa.PrivateKey, []byte) {
	t.Helper()
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pubBytes, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})
	return privKey, pubPEM
}

// signToken creates a signed JWT for testing.
func signToken(t *testing.T, privKey *rsa.PrivateKey, claims usecase.VNPClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(privKey)
	if err != nil {
		t.Fatal(err)
	}
	return signed
}

type mockKeyStore struct {
	result *domain.AuthContext
	err    error
}

func (m *mockKeyStore) ResolveAPIKey(_ context.Context, _ string) (*domain.AuthContext, error) {
	return m.result, m.err
}

type mockPublisher struct{}

func (m *mockPublisher) Publish(_ context.Context, _ string, _ any) error { return nil }

func TestAuthenticateJWT_Valid(t *testing.T) {
	privKey, pubPEM := testKeyPair(t)

	uc, err := usecase.NewAuthUseCase(
		&mockKeyStore{}, &mockPublisher{},
		pubPEM, "vnp-memory", "vnp-api",
		false, testLogger,
	)
	if err != nil {
		t.Fatal(err)
	}

	claims := usecase.VNPClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "vnp-memory",
			Audience:  jwt.ClaimStrings{"vnp-api"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		TenantID: "tenant-abc",
		UserID:   "user-123",
		Roles:    []string{"editor"},
		Scopes:   []string{"read", "write"},
		RateTier: "pro",
	}

	tokenStr := signToken(t, privKey, claims)

	ctx := context.Background()
	authCtx, err := uc.AuthenticateJWT(ctx, "Bearer "+tokenStr)
	if err != nil {
		t.Fatal(err)
	}

	if authCtx.TenantID != "tenant-abc" {
		t.Errorf("TenantID = %q, want %q", authCtx.TenantID, "tenant-abc")
	}
	if authCtx.UserID != "user-123" {
		t.Errorf("UserID = %q, want %q", authCtx.UserID, "user-123")
	}
	if authCtx.RateTier != "pro" {
		t.Errorf("RateTier = %q, want %q", authCtx.RateTier, "pro")
	}
}

func TestAuthenticateJWT_Expired(t *testing.T) {
	privKey, pubPEM := testKeyPair(t)

	uc, err := usecase.NewAuthUseCase(
		&mockKeyStore{}, &mockPublisher{},
		pubPEM, "vnp-memory", "vnp-api",
		false, testLogger,
	)
	if err != nil {
		t.Fatal(err)
	}

	claims := usecase.VNPClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "vnp-memory",
			Audience:  jwt.ClaimStrings{"vnp-api"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
		TenantID: "tenant-abc",
	}

	tokenStr := signToken(t, privKey, claims)
	_, err = uc.AuthenticateJWT(context.Background(), tokenStr)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestAuthenticateJWT_WrongIssuer(t *testing.T) {
	privKey, pubPEM := testKeyPair(t)

	uc, err := usecase.NewAuthUseCase(
		&mockKeyStore{}, &mockPublisher{},
		pubPEM, "vnp-memory", "vnp-api",
		false, testLogger,
	)
	if err != nil {
		t.Fatal(err)
	}

	claims := usecase.VNPClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "wrong-issuer",
			Audience:  jwt.ClaimStrings{"vnp-api"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		TenantID: "tenant-abc",
	}

	tokenStr := signToken(t, privKey, claims)
	_, err = uc.AuthenticateJWT(context.Background(), tokenStr)
	if err == nil {
		t.Fatal("expected error for wrong issuer")
	}
}

func TestAuthenticateJWT_DevMode(t *testing.T) {
	uc, err := usecase.NewAuthUseCase(
		&mockKeyStore{}, &mockPublisher{},
		nil, "vnp-memory", "vnp-api",
		true, testLogger,
	)
	if err != nil {
		t.Fatal(err)
	}

	// Empty token in dev mode should return DevAuthContext
	authCtx, err := uc.AuthenticateJWT(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if authCtx.TenantID != "dev-tenant" {
		t.Errorf("DevMode TenantID = %q, want %q", authCtx.TenantID, "dev-tenant")
	}
}

func TestAuthenticateJWT_MissingTenantClaim(t *testing.T) {
	privKey, pubPEM := testKeyPair(t)

	uc, err := usecase.NewAuthUseCase(
		&mockKeyStore{}, &mockPublisher{},
		pubPEM, "vnp-memory", "vnp-api",
		false, testLogger,
	)
	if err != nil {
		t.Fatal(err)
	}

	claims := usecase.VNPClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "vnp-memory",
			Audience:  jwt.ClaimStrings{"vnp-api"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		// TenantID intentionally empty
	}

	tokenStr := signToken(t, privKey, claims)
	_, err = uc.AuthenticateJWT(context.Background(), tokenStr)
	if err == nil {
		t.Fatal("expected error for missing tenant_id claim")
	}
}

func TestAuthenticateAPIKey_Valid(t *testing.T) {
	expected := &domain.AuthContext{
		TenantID: "tenant-xyz",
		UserID:   "user-456",
		Roles:    []string{"api_key"},
		Scopes:   []string{"*"},
		RateTier: "enterprise",
	}

	uc, _ := usecase.NewAuthUseCase(
		&mockKeyStore{result: expected}, &mockPublisher{},
		nil, "", "",
		false, testLogger,
	)

	authCtx, err := uc.AuthenticateAPIKey(context.Background(), "vnp_test123456")
	if err != nil {
		t.Fatal(err)
	}
	if authCtx.TenantID != "tenant-xyz" {
		t.Errorf("TenantID = %q, want %q", authCtx.TenantID, "tenant-xyz")
	}
}

func TestAuthenticateAPIKey_InvalidPrefix(t *testing.T) {
	uc, _ := usecase.NewAuthUseCase(
		&mockKeyStore{}, &mockPublisher{},
		nil, "", "",
		false, testLogger,
	)

	_, err := uc.AuthenticateAPIKey(context.Background(), "sk_wrong_prefix")
	if err == nil {
		t.Fatal("expected error for invalid prefix")
	}
}
