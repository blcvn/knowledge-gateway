package usecase

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

	"vnp-memory/services/sm-auth/internal/domain/model"
	"vnp-memory/services/sm-auth/internal/usecase/port"
)

type AuthUseCase struct {
	userRepo       port.UserRepository
	jwtPrivateKey  *rsa.PrivateKey
	googleClientID string
}

func NewAuthUseCase(repo port.UserRepository, jwtPrivKeyPEM string, googleClientID string) (*AuthUseCase, error) {
	var rsaPrivKey *rsa.PrivateKey
	if len(jwtPrivKeyPEM) > 0 {
		block, _ := pem.Decode([]byte(jwtPrivKeyPEM))
		if block == nil {
			return nil, fmt.Errorf("failed to decode PEM block for JWT private key")
		}
		
		var err error
		rsaPrivKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			privKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse JWT private key: %v", err)
			}
			var ok bool
			rsaPrivKey, ok = privKey.(*rsa.PrivateKey)
			if !ok {
				return nil, fmt.Errorf("JWT private key is not RSA")
			}
		}
	} else {
		return nil, fmt.Errorf("AUTH_JWT_PRIVATE_KEY is required")
	}

	return &AuthUseCase{
		userRepo:       repo,
		jwtPrivateKey:  rsaPrivKey,
		googleClientID: googleClientID,
	}, nil
}

func (uc *AuthUseCase) Register(ctx context.Context, name, email, password string) (*model.User, string, error) {
	existing, _ := uc.userRepo.GetByEmail(ctx, email)
	if existing != nil {
		return nil, "", errors.New("email already registered")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	user := &model.User{
		ID:           generateID(), // implement id gen
		Email:        email,
		Name:         name,
		PasswordHash: string(hash),
		AuthProvider: "email",
		Role:         "User",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := uc.userRepo.Create(ctx, user); err != nil {
		return nil, "", err
	}

	token, err := uc.generateJWT(user)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

func (uc *AuthUseCase) Login(ctx context.Context, email, password string) (*model.User, string, error) {
	user, err := uc.userRepo.GetByEmail(ctx, email)
	if err != nil || user == nil {
		return nil, "", errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, "", errors.New("invalid credentials")
	}

	token, err := uc.generateJWT(user)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

func (uc *AuthUseCase) LoginWithGoogle(ctx context.Context, idToken string) (*model.User, string, error) {
	payload, err := idtoken.Validate(ctx, idToken, uc.googleClientID)
	if err != nil {
		return nil, "", errors.New("invalid google token")
	}

	email, ok := payload.Claims["email"].(string)
	if !ok {
		return nil, "", errors.New("email not found in google token")
	}

	name, _ := payload.Claims["name"].(string)

	user, err := uc.userRepo.GetByEmail(ctx, email)
	if err != nil || user == nil {
		// Auto-register
		user = &model.User{
			ID:           generateID(),
			Email:        email,
			Name:         name,
			AuthProvider: "google",
			Role:         "User",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		if err := uc.userRepo.Create(ctx, user); err != nil {
			return nil, "", err
		}
	}

	token, err := uc.generateJWT(user)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

func (uc *AuthUseCase) generateJWT(user *model.User) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub":   user.ID,
		"uid":   user.ID,
		"tid":   "default-tenant",
		"email": user.Email,
		"name":  user.Name,
		"roles": []string{user.Role},
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
		"iss":   "vnp-memory",
		"aud":   "vnp-memory-api",
	})
	return token.SignedString(uc.jwtPrivateKey)
}

func generateID() string {
	return "usr_" + time.Now().Format("20060102150405")
}
