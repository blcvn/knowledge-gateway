package grpc

import (
	"context"

	"github.com/vnp-community/vnp-memory/services/ov-storage/internal/usecase/port"
	"google.golang.org/protobuf/types/known/emptypb"
)

// OvFsServiceServer is the gRPC handler for FS operations
type OvFsServiceServer struct {
	fsUC port.FsUseCase
}

func NewOvFsServiceServer(fsUC port.FsUseCase) *OvFsServiceServer {
	return &OvFsServiceServer{fsUC: fsUC}
}

// Implement mock endpoints to satisfy interface
// func (s *OvFsServiceServer) ReadFile(ctx context.Context, req *ReadFileRequest) (*ReadFileResponse, error) { ... }

// OvCryptoServiceServer is the gRPC handler for Crypto operations
type OvCryptoServiceServer struct {
	cryptoUC port.CryptoUseCase
}

func NewOvCryptoServiceServer(cryptoUC port.CryptoUseCase) *OvCryptoServiceServer {
	return &OvCryptoServiceServer{cryptoUC: cryptoUC}
}

// OvResourceServiceServer is the gRPC handler for Resource operations
type OvResourceServiceServer struct {
	resUC port.ResourceUseCase
}

func NewOvResourceServiceServer(resUC port.ResourceUseCase) *OvResourceServiceServer {
	return &OvResourceServiceServer{resUC: resUC}
}
