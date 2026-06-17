package access

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"kg-service/internal/platform/rediscache"
)

var (
	ErrForbidden        = errors.New("forbidden")
	ErrNotFound         = errors.New("not found")
	ErrValidation       = errors.New("validation")
	ErrRequestMalformed = errors.New("request malformed")
)

type Service struct {
	store TenantAppStore
	cache *rediscache.Client
	now   func() time.Time
}

type TenantAppStore interface {
	TenantStore
	AppStore
}

func NewService(store TenantAppStore, cache *rediscache.Client) *Service {
	return &Service{
		store: store,
		cache: cache,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (s *Service) CreateTenant(actor Identity, req TenantCreateRequest) (TenantResponse, error) {
	if !isPlatformAdmin(actor) {
		return TenantResponse{}, ErrForbidden
	}
	if err := validateTenantCreateRequest(req); err != nil {
		return TenantResponse{}, errors.Join(ErrValidation, err)
	}

	now := s.now()
	tenant := Tenant{
		ID:                   newID("tenant"),
		Slug:                 strings.TrimSpace(req.Slug),
		Name:                 strings.TrimSpace(req.Name),
		Status:               "active",
		Tier:                 fallback(req.Tier, "free"),
		DefaultSharingPolicy: "deny_all",
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	s.store.CreateTenant(tenant)
	return tenantToResponse(tenant), nil
}

func (s *Service) GetTenant(actor Identity, tenantID string) (TenantResponse, error) {
	if !canManageTenant(actor, tenantID) {
		return TenantResponse{}, ErrForbidden
	}
	tenant, ok := s.store.GetTenant(tenantID)
	if !ok {
		return TenantResponse{}, ErrNotFound
	}
	return tenantToResponse(tenant), nil
}

func (s *Service) UpdateTenant(actor Identity, tenantID string, req TenantUpdateRequest) (TenantResponse, error) {
	if !canManageTenant(actor, tenantID) {
		return TenantResponse{}, ErrForbidden
	}
	tenant, ok := s.store.GetTenant(tenantID)
	if !ok {
		return TenantResponse{}, ErrNotFound
	}
	if err := validateTenantUpdateRequest(req); err != nil {
		return TenantResponse{}, errors.Join(ErrValidation, err)
	}

	if req.Tier != "" {
		tenant.Tier = req.Tier
	}
	if req.DefaultSharingPolicy != "" {
		tenant.DefaultSharingPolicy = req.DefaultSharingPolicy
	}

	updated, ok := s.store.UpdateTenant(tenant)
	if !ok {
		return TenantResponse{}, ErrNotFound
	}
	return tenantToResponse(updated), nil
}

func (s *Service) SuspendTenant(actor Identity, tenantID string) (TenantResponse, error) {
	if !canManageTenant(actor, tenantID) {
		return TenantResponse{}, ErrForbidden
	}
	tenant, ok := s.store.GetTenant(tenantID)
	if !ok {
		return TenantResponse{}, ErrNotFound
	}
	tenant.Status = "suspended"
	updated, ok := s.store.UpdateTenant(tenant)
	if !ok {
		return TenantResponse{}, ErrNotFound
	}
	return tenantToResponse(updated), nil
}

func (s *Service) CreateApp(actor Identity, tenantID string, req AppCreateRequest) (AppResponse, error) {
	if !canManageTenant(actor, tenantID) {
		return AppResponse{}, ErrForbidden
	}
	if _, ok := s.store.GetTenant(tenantID); !ok {
		return AppResponse{}, ErrNotFound
	}
	if err := validateAppCreateRequest(req); err != nil {
		return AppResponse{}, errors.Join(ErrValidation, err)
	}

	apiKey, apiKeyHash, err := newAPIKey()
	if err != nil {
		return AppResponse{}, err
	}

	now := s.now()
	app := App{
		ID:           newID("app"),
		TenantID:     tenantID,
		Slug:         strings.TrimSpace(req.Slug),
		Name:         strings.TrimSpace(req.Name),
		Type:         req.Type,
		APIKeyHash:   apiKeyHash,
		APIKeyPrefix: apiKeyPrefix(apiKey),
		Status:       "active",
		CreatedAt:    now,
	}
	s.store.CreateApp(app)
	return appToResponse(app, apiKey), nil
}

func (s *Service) RotateAppKey(actor Identity, tenantID, appID string) (RotateKeyResponse, error) {
	if !canManageTenant(actor, tenantID) {
		return RotateKeyResponse{}, ErrForbidden
	}
	app, ok := s.store.GetAppByID(appID)
	if !ok || app.TenantID != tenantID {
		return RotateKeyResponse{}, ErrNotFound
	}

	oldHash := app.APIKeyHash
	apiKey, apiKeyHash, err := newAPIKey()
	if err != nil {
		return RotateKeyResponse{}, err
	}

	rotatedAt := s.now()
	app.APIKeyHash = apiKeyHash
	app.APIKeyPrefix = apiKeyPrefix(apiKey)
	_, ok = s.store.UpdateApp(app)
	if !ok {
		return RotateKeyResponse{}, ErrNotFound
	}

	s.cache.Delete("apikey:" + oldHash)
	return RotateKeyResponse{
		APIKey:    apiKey,
		RotatedAt: rotatedAt,
	}, nil
}

func (s *Service) ListApps(actor Identity, tenantID string) ([]AppResponse, error) {
	if !canManageTenant(actor, tenantID) {
		return nil, ErrForbidden
	}
	if _, ok := s.store.GetTenant(tenantID); !ok {
		return nil, ErrNotFound
	}

	apps := s.store.ListAppsByTenant(tenantID)
	result := make([]AppResponse, 0, len(apps))
	for _, app := range apps {
		result = append(result, appToResponse(app, ""))
	}
	return result, nil
}

func (s *Service) RevokeApp(actor Identity, tenantID, appID string) (AppResponse, error) {
	if !canManageTenant(actor, tenantID) {
		return AppResponse{}, ErrForbidden
	}
	app, ok := s.store.GetAppByID(appID)
	if !ok || app.TenantID != tenantID {
		return AppResponse{}, ErrNotFound
	}

	now := s.now()
	app.Status = "revoked"
	app.RevokedAt = &now
	updated, ok := s.store.UpdateApp(app)
	if !ok {
		return AppResponse{}, ErrNotFound
	}

	s.cache.Delete("apikey:" + updated.APIKeyHash)
	s.cache.Delete("acl:" + updated.TenantID + ":" + updated.ID)
	return appToResponse(updated, ""), nil
}

func validateTenantCreateRequest(req TenantCreateRequest) error {
	if strings.TrimSpace(req.Slug) == "" || strings.TrimSpace(req.Name) == "" {
		return errors.New("slug and name are required")
	}
	if req.Tier != "" && !isAllowed(req.Tier, []string{"free", "pro", "enterprise"}) {
		return fmt.Errorf("invalid tier: %s", req.Tier)
	}
	return nil
}

func validateTenantUpdateRequest(req TenantUpdateRequest) error {
	if req.Tier != "" && !isAllowed(req.Tier, []string{"free", "pro", "enterprise"}) {
		return fmt.Errorf("invalid tier: %s", req.Tier)
	}
	if req.DefaultSharingPolicy != "" && !isAllowed(req.DefaultSharingPolicy, []string{"deny_all", "share_within_tenant_read"}) {
		return fmt.Errorf("invalid default_sharing_policy: %s", req.DefaultSharingPolicy)
	}
	return nil
}

func validateAppCreateRequest(req AppCreateRequest) error {
	if strings.TrimSpace(req.Slug) == "" || strings.TrimSpace(req.Name) == "" {
		return errors.New("slug and name are required")
	}
	if !isAllowed(req.Type, []string{"agent_consumer", "ingestion_producer", "admin_tool", "hybrid"}) {
		return fmt.Errorf("invalid app type: %s", req.Type)
	}
	return nil
}

func isPlatformAdmin(actor Identity) bool {
	return actor.TenantID == PlatformTenantID && (actor.AppType == "admin_tool" || actor.AppType == "hybrid")
}

func canManageTenant(actor Identity, tenantID string) bool {
	if isPlatformAdmin(actor) {
		return true
	}
	return actor.TenantID == tenantID && (actor.AppType == "admin_tool" || actor.AppType == "hybrid")
}

func tenantToResponse(tenant Tenant) TenantResponse {
	return TenantResponse{
		ID:                   tenant.ID,
		Slug:                 tenant.Slug,
		Name:                 tenant.Name,
		Status:               tenant.Status,
		Tier:                 tenant.Tier,
		DefaultSharingPolicy: tenant.DefaultSharingPolicy,
		CreatedAt:            tenant.CreatedAt,
		UpdatedAt:            tenant.UpdatedAt,
	}
}

func appToResponse(app App, apiKey string) AppResponse {
	return AppResponse{
		ID:        app.ID,
		TenantID:  app.TenantID,
		Slug:      app.Slug,
		Name:      app.Name,
		Type:      app.Type,
		Status:    app.Status,
		APIKey:    apiKey,
		CreatedAt: app.CreatedAt,
		RevokedAt: app.RevokedAt,
	}
}

func newID(prefix string) string {
	return prefix + "_" + time.Now().UTC().Format("20060102150405.000000000")
}

func newAPIKey() (string, string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	token := "kgsk_live_" + hex.EncodeToString(buf)
	return token, APIKeyHash(token), nil
}

func apiKeyPrefix(apiKey string) string {
	if len(apiKey) < 8 {
		return apiKey
	}
	return apiKey[:8]
}

func fallback(value, defaultValue string) string {
	if strings.TrimSpace(value) == "" {
		return defaultValue
	}
	return value
}

func isAllowed(value string, allowed []string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
