package biz

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

const (
	DefaultRequestsPerMinute = int64(1000)
	quotaTypeRequestsPerMin  = "requests_per_minute"
)

var (
	ErrAPIKeyNotFound = errors.New("api key not found")
	ErrAPIKeyRevoked  = errors.New("api key revoked")
	ErrAPIKeyExpired  = errors.New("api key expired")
)

type RegistryRepo interface {
	CreateApp(ctx context.Context, app *App) error
	GetApp(ctx context.Context, appID string) (*App, error)
	ListApps(ctx context.Context) ([]*App, error)
	CreateAPIKey(ctx context.Context, key *APIKey) error
	GetAPIKeyByHash(ctx context.Context, keyHash string) (*APIKey, error)
	RevokeAPIKey(ctx context.Context, keyHash string) error
	GetQuota(ctx context.Context, appID, quotaType string) (*Quota, error)
}

type RegistryUsecase struct {
	repo RegistryRepo
	log  *log.Helper
}

func NewRegistryUsecase(repo RegistryRepo, logger log.Logger) *RegistryUsecase {
	return &RegistryUsecase{
		repo: repo,
		log:  log.NewHelper(logger),
	}
}

func (uc *RegistryUsecase) CreateApp(ctx context.Context, appName, description, owner string) (*App, error) {
	app := &App{
		AppID:       uuid.NewString(),
		AppName:     appName,
		Description: description,
		Owner:       owner,
		Status:      "ACTIVE",
	}
	if err := uc.repo.CreateApp(ctx, app); err != nil {
		return nil, err
	}
	return app, nil
}

func (uc *RegistryUsecase) GetApp(ctx context.Context, appID string) (*App, error) {
	return uc.repo.GetApp(ctx, appID)
}

func (uc *RegistryUsecase) ListApps(ctx context.Context) ([]*App, error) {
	return uc.repo.ListApps(ctx)
}

func (uc *RegistryUsecase) IssueAPIKey(ctx context.Context, appID, name, scopes string, ttlSeconds int64) (rawKey string, key *APIKey, err error) {
	if _, err = uc.repo.GetApp(ctx, appID); err != nil {
		return "", nil, err
	}

	rawKey, err = generateRawAPIKey()
	if err != nil {
		return "", nil, err
	}

	hash := HashAPIKey(rawKey)
	key = &APIKey{
		KeyHash:   hash,
		AppID:     appID,
		KeyPrefix: keyPrefix(rawKey),
		Name:      name,
		Scopes:    scopes,
	}

	if ttlSeconds > 0 {
		expires := time.Now().UTC().Add(time.Duration(ttlSeconds) * time.Second)
		key.ExpiresAt = &expires
	}

	if err := uc.repo.CreateAPIKey(ctx, key); err != nil {
		return "", nil, err
	}
	return rawKey, key, nil
}

func (uc *RegistryUsecase) RevokeAPIKey(ctx context.Context, keyHash string) error {
	return uc.repo.RevokeAPIKey(ctx, normalizeKeyHash(keyHash))
}

func (uc *RegistryUsecase) ValidateAPIKey(ctx context.Context, rawAPIKey string) (*APIKey, error) {
	key, err := uc.repo.GetAPIKeyByHash(ctx, HashAPIKey(rawAPIKey))
	if err != nil {
		return nil, ErrAPIKeyNotFound
	}
	if key.IsRevoked {
		return nil, ErrAPIKeyRevoked
	}
	if key.ExpiresAt != nil && key.ExpiresAt.Before(time.Now().UTC()) {
		return nil, ErrAPIKeyExpired
	}
	return key, nil
}

func (uc *RegistryUsecase) GetQuotaLimit(ctx context.Context, appID string, fallback int64) (int64, error) {
	quota, err := uc.repo.GetQuota(ctx, appID, quotaTypeRequestsPerMin)
	if err != nil {
		return fallback, nil
	}
	if quota.Limit <= 0 {
		return fallback, nil
	}
	return quota.Limit, nil
}

func HashAPIKey(rawAPIKey string) string {
	digest := sha256.Sum256([]byte(rawAPIKey))
	return "sha256_" + hex.EncodeToString(digest[:])
}

func keyPrefix(rawKey string) string {
	if len(rawKey) <= 10 {
		return rawKey
	}
	return rawKey[:10]
}

func normalizeKeyHash(hash string) string {
	if strings.HasPrefix(hash, "sha256_") {
		return hash
	}
	return "sha256_" + hash
}

func generateRawAPIKey() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "kgs_ak_" + hex.EncodeToString(b), nil
}
