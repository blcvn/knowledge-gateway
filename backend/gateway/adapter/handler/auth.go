// Package handler — Auth handler for VNP Console login/logout/refresh/me.
// Implements TASK-BE-002: Console Auth endpoints.
package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/vnp-community/vnp-memory/gateway/infra/middleware"
	"github.com/vnp-community/vnp-memory/gateway/usecase"
	"github.com/vnp-community/vnp-memory/gateway/usecase/port"
)

// AuthHandler handles /v1/auth/* routes.
// Khi pool != nil: xử lý login/logout/refresh/me trực tiếp với PostgreSQL.
// Khi pool == nil: forward đến sm-auth service (fallback).
type AuthHandler struct {
	registry port.ServiceRegistry
	pool     *pgxpool.Pool    // Optional — nếu nil thì forward đến sm-auth
	authUC   *usecase.AuthUseCase // For JWT signing (if jwtPrivKey provided)
	logger   *slog.Logger
}

// NewAuthHandler creates a new AuthHandler (registry-only, backward compatible).
func NewAuthHandler(registry port.ServiceRegistry, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{
		registry: registry,
		logger:   logger,
	}
}

// NewAuthHandlerWithDB creates an AuthHandler with direct DB access.
// Dùng khi Console Auth cần login trực tiếp với PostgreSQL console_users table.
func NewAuthHandlerWithDB(registry port.ServiceRegistry, pool *pgxpool.Pool, authUC *usecase.AuthUseCase, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{
		registry: registry,
		pool:     pool,
		authUC:   authUC,
		logger:   logger,
	}
}

// POST /v1/auth/login — Email + password login, trả về access_token + refresh_token.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if h.pool == nil {
		// Fallback: forward sang sm-auth service
		ForwardToService(h.registry, "sm-auth", h.logger)(w, r)
		return
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" || req.Password == "" {
		writeJSONError(w, "email và password là bắt buộc", "INVALID_REQUEST", 400)
		return
	}

	// 1. Tìm user theo email
	var user struct {
		ID           string
		Name         string
		Email        string
		PasswordHash string
		Role         string
		TenantID     string
		AvatarURL    *string
	}
	err := h.pool.QueryRow(r.Context(),
		`SELECT id, name, email, password_hash, role, tenant_id::text, avatar_url
		 FROM console_users WHERE email = $1 AND is_active = true`,
		req.Email,
	).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash,
		&user.Role, &user.TenantID, &user.AvatarURL)
	if err != nil {
		h.logger.Debug("login: user not found", "email", req.Email, "error", err)
		writeJSONError(w, "Email hoặc mật khẩu không đúng", "AUTH_INVALID_CREDENTIALS", 401)
		return
	}

	// 2. Verify bcrypt password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		writeJSONError(w, "Email hoặc mật khẩu không đúng", "AUTH_INVALID_CREDENTIALS", 401)
		return
	}

	// 3. Tạo access token (JWT — dùng authUC nếu có, hoặc tạo simple token)
	accessToken, expiresIn := h.signAccessToken(user.ID, user.Role, user.TenantID)

	// 4. Tạo refresh token (32 bytes random, lưu SHA-256 hash)
	rawRefresh, err := generateToken(32)
	if err != nil {
		h.logger.Error("login: failed to generate refresh token", "error", err)
		writeJSONError(w, "Lỗi máy chủ", "INTERNAL_ERROR", 500)
		return
	}
	hash := sha256.Sum256([]byte(rawRefresh))
	tokenHash := hex.EncodeToString(hash[:])
	expiresRefresh := time.Now().UTC().Add(30 * 24 * time.Hour)

	_, err = h.pool.Exec(r.Context(),
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		 VALUES ($1, $2, $3)`,
		user.ID, tokenHash, expiresRefresh,
	)
	if err != nil {
		h.logger.Error("login: failed to store refresh token", "error", err)
		writeJSONError(w, "Lỗi máy chủ", "INTERNAL_ERROR", 500)
		return
	}

	// 5. Trả về response
	writeJSON(w, 200, map[string]any{
		"access_token":  accessToken,
		"refresh_token": rawRefresh,
		"expires_in":    expiresIn,
		"token_type":    "Bearer",
		"user": map[string]any{
			"id":         user.ID,
			"name":       user.Name,
			"email":      user.Email,
			"role":       user.Role,
			"tenant_id":  user.TenantID,
			"avatar_url": user.AvatarURL,
		},
	})
}

// POST /v1/auth/logout — Revoke refresh token.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if h.pool == nil {
		w.WriteHeader(204)
		return
	}

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	if req.RefreshToken != "" {
		hash := sha256.Sum256([]byte(req.RefreshToken))
		tokenHash := hex.EncodeToString(hash[:])
		_, _ = h.pool.Exec(r.Context(),
			`UPDATE refresh_tokens SET revoked = true WHERE token_hash = $1`,
			tokenHash,
		)
	}
	w.WriteHeader(204)
}

// POST /v1/auth/refresh — Dùng refresh token để lấy access token mới.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	if h.pool == nil {
		writeJSONError(w, "Refresh không khả dụng", "NOT_SUPPORTED", 501)
		return
	}

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		writeJSONError(w, "refresh_token là bắt buộc", "INVALID_REQUEST", 400)
		return
	}

	hash := sha256.Sum256([]byte(req.RefreshToken))
	tokenHash := hex.EncodeToString(hash[:])

	var userID, role, tenantID string
	var expiresAt time.Time
	var revoked bool
	err := h.pool.QueryRow(r.Context(),
		`SELECT rt.user_id::text, cu.role, cu.tenant_id::text, rt.expires_at, rt.revoked
		 FROM refresh_tokens rt
		 JOIN console_users cu ON rt.user_id = cu.id
		 WHERE rt.token_hash = $1`,
		tokenHash,
	).Scan(&userID, &role, &tenantID, &expiresAt, &revoked)

	if err != nil || revoked || time.Now().After(expiresAt) {
		writeJSONError(w, "Refresh token không hợp lệ hoặc đã hết hạn", "AUTH_TOKEN_INVALID", 401)
		return
	}

	accessToken, expiresIn := h.signAccessToken(userID, role, tenantID)
	writeJSON(w, 200, map[string]any{
		"access_token": accessToken,
		"expires_in":   expiresIn,
	})
}

// GET /v1/auth/me — Trả về thông tin user hiện tại (yêu cầu JWT).
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	auth, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		writeJSONError(w, "Chưa xác thực", "UNAUTHENTICATED", 401)
		return
	}

	if h.pool == nil {
		// Trả về thông tin từ JWT context
		writeJSON(w, 200, map[string]any{
			"id":        auth.UserID,
			"tenant_id": auth.TenantID,
			"roles":     auth.Roles,
		})
		return
	}

	var user struct {
		ID        string
		Name      string
		Email     string
		Role      string
		TenantID  string
		AvatarURL *string
	}
	err := h.pool.QueryRow(r.Context(),
		`SELECT id::text, name, email, role, tenant_id::text, avatar_url
		 FROM console_users WHERE id = $1`,
		auth.UserID,
	).Scan(&user.ID, &user.Name, &user.Email, &user.Role, &user.TenantID, &user.AvatarURL)
	if err != nil {
		writeJSONError(w, "User không tồn tại", "NOT_FOUND", 404)
		return
	}

	writeJSON(w, 200, map[string]any{
		"id":         user.ID,
		"name":       user.Name,
		"email":      user.Email,
		"role":       user.Role,
		"tenant_id":  user.TenantID,
		"avatar_url": user.AvatarURL,
	})
}

// POST /v1/auth/sso/google — Google OAuth redirect (forward sang sm-auth).
func (h *AuthHandler) LoginWithGoogle(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "sm-auth", h.logger)(w, r)
}

// POST /v1/auth/register — User registration (forward sang sm-auth hoặc local).
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "sm-auth", h.logger)(w, r)
}

// signAccessToken tạo opaque access token (48 bytes random = 96 hex chars).
// Trong production, nên thay bằng RS256 JWT từ authUC.
// Token đủ entropy để dùng làm bearer token tạm thời.
func (h *AuthHandler) signAccessToken(userID, role, tenantID string) (string, int) {
	const ttl = 3600 // 1 giờ

	// Log để trace
	h.logger.Debug("signAccessToken", "user", userID, "role", role, "tenant", tenantID)

	// Tạo random token (48 bytes = 96 hex chars)
	token, err := generateToken(48)
	if err != nil {
		// Fallback: dùng hash của inputs + timestamp
		data := userID + ":" + role + ":" + tenantID + ":" + hex.EncodeToString([]byte(time.Now().String()))
		sum := sha256.Sum256([]byte(data))
		return hex.EncodeToString(sum[:]), ttl
	}
	return token, ttl
}

// generateToken tạo random hex string với n bytes entropy.
func generateToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// writeJSON writes JSON response.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// writeJSONError writes a structured JSON error response.
func writeJSONError(w http.ResponseWriter, message, code string, status int) {
	writeJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}
