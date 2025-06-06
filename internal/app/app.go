package app

import (
	grpcapp "Service/internal/app/grpc"
	"Service/internal/services/auth"
	"Service/internal/services/follow"
	"Service/internal/services/userinfo"
	"Service/internal/storage/sqlite"
	"log/slog"
	"time"
)

type App struct {
	GRPCApp *grpcapp.App
}

func New(
	log *slog.Logger,
	storagePath string,
	secret string,
	tokenTTL time.Duration,
	refreshTTL time.Duration,
	port int,
	timeout time.Duration,
) *App {

	st := sqlite.New(storagePath)

	authsrvc := auth.New(log, st, st, st, secret, tokenTTL, refreshTTL)
	usrInfo := userinfo.New(log, st)
	fllw := follow.New(log, st, st, st)
	gRPCApp := grpcapp.New(log, port, timeout, authsrvc, usrInfo, fllw)

	return &App{
		GRPCApp: gRPCApp,
	}
}
