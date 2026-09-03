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
