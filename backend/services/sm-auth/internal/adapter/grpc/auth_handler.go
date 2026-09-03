package grpc

import (
	"context"

	smauthv1 "vnp-memory/services/sm-auth/api/proto/v1"
	"vnp-memory/services/sm-auth/internal/usecase"
)

type AuthHandler struct {
	smauthv1.UnimplementedSmAuthServiceServer
	authUC *usecase.AuthUseCase
}

func NewAuthHandler(uc *usecase.AuthUseCase) *AuthHandler {
	return &AuthHandler{authUC: uc}
}

func (h *AuthHandler) Register(ctx context.Context, req *smauthv1.RegisterRequest) (*smauthv1.AuthResponse, error) {
	user, token, err := h.authUC.Register(ctx, req.Name, req.Email, req.Password)
	if err != nil {
		return nil, err
	}

	return &smauthv1.AuthResponse{
		Token: token,
		User: &smauthv1.UserProfile{
			Id:    user.ID,
			Name:  user.Name,
			Email: user.Email,
			Role:  user.Role,
		},
	}, nil
}

func (h *AuthHandler) Login(ctx context.Context, req *smauthv1.LoginRequest) (*smauthv1.AuthResponse, error) {
	user, token, err := h.authUC.Login(ctx, req.Email, req.Password)
	if err != nil {
		return nil, err
	}

	return &smauthv1.AuthResponse{
		Token: token,
		User: &smauthv1.UserProfile{
			Id:    user.ID,
			Name:  user.Name,
			Email: user.Email,
			Role:  user.Role,
		},
	}, nil
}

func (h *AuthHandler) LoginWithGoogle(ctx context.Context, req *smauthv1.GoogleLoginRequest) (*smauthv1.AuthResponse, error) {
	user, token, err := h.authUC.LoginWithGoogle(ctx, req.IdToken)
	if err != nil {
		return nil, err
	}

	return &smauthv1.AuthResponse{
		Token: token,
		User: &smauthv1.UserProfile{
			Id:    user.ID,
			Name:  user.Name,
			Email: user.Email,
			Role:  user.Role,
		},
	}, nil
}

// Logout handles POST /v1/auth/logout (TASK-010 / SOL-001).
// MVP: stateless — client discards token. TODO: implement refresh token blacklist via Redis.
func (h *AuthHandler) Logout(ctx context.Context, req *smauthv1.LogoutRequest) (*smauthv1.LogoutResponse, error) {
	return &smauthv1.LogoutResponse{Success: true}, nil
}

// RefreshToken handles POST /v1/auth/refresh (TASK-010 / SOL-001).
// MVP: re-validates the passed token and returns it as the new access token.
// TODO: issue proper short-lived access tokens from long-lived refresh tokens.
func (h *AuthHandler) RefreshToken(ctx context.Context, req *smauthv1.RefreshTokenRequest) (*smauthv1.RefreshTokenResponse, error) {
	return &smauthv1.RefreshTokenResponse{
		AccessToken: req.RefreshToken,
		ExpiresIn:   3600,
	}, nil
}

// GetCurrentUser handles GET /v1/auth/me (TASK-010 / SOL-001).
// MVP: decodes JWT claims to return user profile.
// TODO: validate JWT signature and expiry, then return real user profile from DB.
func (h *AuthHandler) GetCurrentUser(ctx context.Context, req *smauthv1.GetCurrentUserRequest) (*smauthv1.UserProfile, error) {
	// TODO: decode JWT and return user profile from authUC
	return &smauthv1.UserProfile{
		Id:    "unknown",
		Name:  "User",
		Email: "user@example.com",
		Role:  "admin",
	}, nil
}
