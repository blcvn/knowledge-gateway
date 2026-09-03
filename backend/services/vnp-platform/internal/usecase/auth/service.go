// Package auth implements the auth usecase for vnp-platform.
//
// Absorbed from: sm-auth (MERGE-P1-T1)
// Replaces: InMemoryUserRepository with PGAuthUserRepository
package auth

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/idtoken"

	"github.com/vnp-community/vnp-memory/services/vnp-platform/internal/domain/auth"
)

// AuthRepository is the output port for auth persistence.
type AuthRepository interface {
	FindByEmail(ctx context.Context, email string) (*auth.AuthUser, error)
	FindByProviderID(ctx context.Context, provider, providerID string) (*auth.AuthUser, error)
	Create(ctx context.Context, user *auth.AuthUser) error
	Update(ctx context.Context, user *auth.AuthUser) error
}

// AuthService implements JWT+SSO authentication.
type AuthService struct {
	repo           AuthRepository
	jwtPrivateKey  *rsa.PrivateKey
	googleClientID string
}

// NewAuthService creates an AuthService.
// jwtPrivKeyPEM must be a PEM-encoded RSA private key (PKCS1 or PKCS8).
// Returns error if AUTH_JWT_PRIVATE_KEY is empty or invalid.
func NewAuthService(repo AuthRepository, jwtPrivKeyPEM string, googleClientID string) (*AuthService, error) {
	if len(jwtPrivKeyPEM) == 0 {
		return nil, fmt.Errorf("AUTH_JWT_PRIVATE_KEY is required")
	}

	block, _ := pem.Decode([]byte(jwtPrivKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block for JWT private key")
	}

	var rsaPrivKey *rsa.PrivateKey
	// Try PKCS1 first, then PKCS8
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		pkcs8Key, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("failed to parse JWT private key (tried PKCS1 and PKCS8): %v / %v", err, err2)
		}
		var ok bool
		rsaPrivKey, ok = pkcs8Key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("JWT private key is not RSA")
		}
	} else {
		rsaPrivKey = key
	}

	return &AuthService{
		repo:           repo,
		jwtPrivateKey:  rsaPrivKey,
		googleClientID: googleClientID,
	}, nil
}

// Register creates a new user with email+password and returns a JWT.
func (s *AuthService) Register(ctx context.Context, name, email, password string) (*auth.AuthUser, *auth.AuthToken, error) {
	existing, _ := s.repo.FindByEmail(ctx, email)
	if existing != nil {
		return nil, nil, errors.New("email already registered")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, fmt.Errorf("hash password: %w", err)
	}

	user := &auth.AuthUser{
		ID:           generateID(),
		Email:        email,
		Name:         name,
		PasswordHash: string(hash),
		AuthProvider: "email",
		Role:         "user",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, nil, fmt.Errorf("create user: %w", err)
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, nil, fmt.Errorf("generate token: %w", err)
	}

	return user, token, nil
}

// Login authenticates with email+password and returns a JWT.
func (s *AuthService) Login(ctx context.Context, email, password string) (*auth.AuthUser, *auth.AuthToken, error) {
	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil || user == nil {
		return nil, nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, nil, errors.New("invalid credentials")
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, nil, fmt.Errorf("generate token: %w", err)
	}

	return user, token, nil
}

// LoginWithGoogle validates a Google ID token and upserts user, returning JWT.
func (s *AuthService) LoginWithGoogle(ctx context.Context, idToken string) (*auth.AuthUser, *auth.AuthToken, error) {
	payload, err := idtoken.Validate(ctx, idToken, s.googleClientID)
	if err != nil {
		return nil, nil, errors.New("invalid google token")
	}

	email, ok := payload.Claims["email"].(string)
	if !ok {
		return nil, nil, errors.New("email not found in google token")
	}
	name, _ := payload.Claims["name"].(string)
	sub, _ := payload.Claims["sub"].(string)

	// Find existing or auto-register
	user, _ := s.repo.FindByEmail(ctx, email)
	if user == nil {
		user, _ = s.repo.FindByProviderID(ctx, "google", sub)
	}
	if user == nil {
		user = &auth.AuthUser{
			ID:             generateID(),
			Email:          email,
			Name:           name,
			AuthProvider:   "google",
			AuthProviderID: sub,
			Role:           "user",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		if err := s.repo.Create(ctx, user); err != nil {
			return nil, nil, fmt.Errorf("create google user: %w", err)
		}
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, nil, fmt.Errorf("generate token: %w", err)
	}

	return user, token, nil
}

// generateToken creates a signed RS256 JWT for the given user.
func (s *AuthService) generateToken(user *auth.AuthUser) (*auth.AuthToken, error) {
	expiry := time.Now().Add(24 * time.Hour)
	claims := jwt.MapClaims{
		"sub":   user.ID,
		"uid":   user.ID,
		"tid":   "default",
		"email": user.Email,
		"name":  user.Name,
		"role":  user.Role,
		"roles": []string{user.Role},
		"exp":   expiry.Unix(),
		"iss":   "vnp-platform",
		"aud":   "vnp-api",
	}

	tokenStr, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(s.jwtPrivateKey)
	if err != nil {
		return nil, err
	}

	return &auth.AuthToken{
		AccessToken: tokenStr,
		TokenType:   "Bearer",
		ExpiresAt:   expiry,
		UserID:      user.ID,
		Email:       user.Email,
		Role:        user.Role,
	}, nil
}

func generateID() string {
	return "usr_" + fmt.Sprintf("%d", time.Now().UnixNano())
}
