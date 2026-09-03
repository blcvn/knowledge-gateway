package usecase

import (
	"context"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/vnp-community/vnp-memory/gateway/internal/domain"
	"github.com/vnp-community/vnp-memory/gateway/internal/usecase/port"
)

// AuthUseCase handles JWT and API key authentication.
type AuthUseCase struct {
	keyStore   port.KeyStore
	publisher  port.EventPublisher
	rsaPubKey  *rsa.PublicKey
	issuer     string
	audience   string
	devMode    bool
	logger     *slog.Logger
}

// NewAuthUseCase creates a new AuthUseCase.
func NewAuthUseCase(
	keyStore port.KeyStore,
	publisher port.EventPublisher,
	jwtPubKeyPEM []byte,
	issuer, audience string,
	devMode bool,
	logger *slog.Logger,
) (*AuthUseCase, error) {
	uc := &AuthUseCase{
		keyStore:  keyStore,
		publisher: publisher,
		issuer:    issuer,
		audience:  audience,
		devMode:   devMode,
		logger:    logger,
	}

	// Parse RSA public key if provided
	if len(jwtPubKeyPEM) > 0 {
		block, _ := pem.Decode(jwtPubKeyPEM)
		if block == nil {
			return nil, fmt.Errorf("failed to decode PEM block for JWT public key")
		}
		pub, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse JWT public key: %w", err)
		}
		rsaPub, ok := pub.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("JWT public key is not RSA")
		}
		uc.rsaPubKey = rsaPub
		logger.Info("JWT RS256 public key loaded")
	} else if !devMode {
		logger.Warn("no JWT public key provided and dev mode is disabled")
	}

	return uc, nil
}

// DevAuthContext is the default AuthContext used when AUTH_DEV_MODE=true.
var DevAuthContext = &domain.AuthContext{
	TenantID: "dev-tenant",
	UserID:   "dev-user",
	Roles:    []string{"admin"},
	Scopes:   []string{"*"},
	RateTier: domain.RateTierEnterprise,
}

// VNPClaims extends jwt.RegisteredClaims with VNP-specific fields.
type VNPClaims struct {
	jwt.RegisteredClaims
	TenantID string   `json:"tid"`
	UserID   string   `json:"uid"`
	Roles    []string `json:"roles"`
	Scopes   []string `json:"scopes"`
	RateTier string   `json:"rate_tier"`
}

// AuthenticateJWT validates a JWT RS256 token and extracts tenant/user claims.
func (uc *AuthUseCase) AuthenticateJWT(ctx context.Context, tokenStr string) (*domain.AuthContext, error) {
	if uc.devMode && tokenStr == "" {
		return DevAuthContext, nil
	}

	if tokenStr == "" {
		return nil, domain.ErrUnauthenticated
	}

	// Strip "Bearer " prefix
	tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")
	tokenStr = strings.TrimSpace(tokenStr)

	if uc.rsaPubKey == nil {
		if uc.devMode {
			return DevAuthContext, nil
		}
		return nil, domain.ErrUnauthenticated.WithMessage("JWT validation unavailable: no public key")
	}

	// Parse and validate token
	token, err := jwt.ParseWithClaims(tokenStr, &VNPClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method is RS256
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return uc.rsaPubKey, nil
	},
		jwt.WithIssuer(uc.issuer),
		jwt.WithAudience(uc.audience),
		jwt.WithLeeway(30*time.Second),
		jwt.WithIssuedAt(),
	)

	if err != nil {
		uc.logger.Debug("JWT validation failed", "error", err)

		// Publish auth failure event
		uc.publisher.Publish(ctx, domain.SubjectAuthFailed, domain.AuthFailed{
			Reason:    "jwt_invalid: " + err.Error(),
			Timestamp: time.Now(),
		})

		return nil, domain.ErrUnauthenticated.WithMessage("invalid JWT: " + err.Error())
	}

	claims, ok := token.Claims.(*VNPClaims)
	if !ok || !token.Valid {
		return nil, domain.ErrUnauthenticated.WithMessage("invalid token claims")
	}

	// Validate required claims
	if claims.TenantID == "" {
		return nil, domain.ErrUnauthenticated.WithMessage("missing tenant_id claim")
	}

	rateTier := claims.RateTier
	if rateTier == "" {
		rateTier = domain.RateTierFree
	}

	authCtx := &domain.AuthContext{
		TenantID: claims.TenantID,
		UserID:   claims.UserID,
		Roles:    claims.Roles,
		Scopes:   claims.Scopes,
		RateTier: rateTier,
	}

	uc.logger.Debug("JWT authenticated",
		"tenant", authCtx.TenantID,
		"user", authCtx.UserID,
		"roles", authCtx.Roles,
	)

	return authCtx, nil
}

// AuthenticateAPIKey resolves an API key to an AuthContext.
func (uc *AuthUseCase) AuthenticateAPIKey(ctx context.Context, key string) (*domain.AuthContext, error) {
	if !strings.HasPrefix(key, "vnp_") {
		return nil, domain.ErrUnauthenticated.WithMessage("invalid API key format")
	}

	// Hash the key for lookup (SHA-256)
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(key)))

	authCtx, err := uc.keyStore.ResolveAPIKey(ctx, hash)
	if err != nil {
		uc.logger.Warn("api key resolution failed", "prefix", key[:8], "error", err)

		// Publish auth failure event
		uc.publisher.Publish(ctx, domain.SubjectAuthFailed, domain.AuthFailed{
			Reason:    "api_key_invalid",
			Timestamp: time.Now(),
		})

		return nil, domain.ErrUnauthenticated.WithMessage("invalid or revoked API key")
	}

	uc.logger.Debug("API key authenticated",
		"tenant", authCtx.TenantID,
		"key_prefix", key[:8],
	)

	return authCtx, nil
}

// IsDevMode returns whether authentication is in development bypass mode.
func (uc *AuthUseCase) IsDevMode() bool {
	return uc.devMode
}
