package grpcapp

import (
	"Service/internal/domain/models"
	grpcauth "Service/internal/grpc/auth"
	grpcfollow "Service/internal/grpc/follow"
	grpcusrinfo "Service/internal/grpc/userinfo"
	"Service/internal/lib/logger/sl"
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type App struct {
	log     *slog.Logger
	gRPCSrv *grpc.Server
	port    int
}

type Auth interface {
	Login(
		ctx context.Context,
		login, password string,
	) (models.TokensPair, error)
	SignUp(
		ctx context.Context,
		login, email, password string,
	) (models.TokensPair, error)
	UpdateTokens(
		ctx context.Context,
		refreshToken string,
	) (models.TokensPair, error)
}

type UserInfo interface {
	User(ctx context.Context, uuid int) (models.User, error)
	Users(ctx context.Context, uuid []int) ([]models.User, error)
	UsersExist(ctx context.Context, uuid []int) (bool, error)
	UsersByLogin(ctx context.Context, login string) ([]models.User, error)
}

type FollowProvider interface {
	Follow(
		ctx context.Context,
		src, target int,
	) error
	Unfollow(
		ctx context.Context,
		src, target int,
	) error
	Followers(
		ctx context.Context,
		uuid int,
	) ([]models.User, error)
	Followees(
		ctx context.Context,
		uuid int,
	) ([]models.User, error)
}

// New
func New(
	log *slog.Logger,
	port int,
	timeout time.Duration,
	auth Auth,
	usrInfo UserInfo,
	followProvider FollowProvider,
) *App {
	recoveryOpts := []recovery.Option{
		recovery.WithRecoveryHandler(
			func(p interface{}) (err error) {
				log.Error("recovering from a panic", slog.Any("panic", p))

				return status.Error(codes.Internal, "internal error")
			},
		),
	}
	loggingOpts := []logging.Option{
		logging.WithLogOnEvents(
			logging.PayloadReceived, logging.PayloadSent,
		),
	}
	_ = loggingOpts
	grpcsrv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			recovery.UnaryServerInterceptor(recoveryOpts...),
			// this part is necessary for logging.
			// I commented it because logs are to large and unreadable
			//
			// logging.UnaryServerInterceptor(
			// 	logInterceptor(log),
			// 	loggingOpts...,
			// ),
		),
	)

	grpcauth.Register(grpcsrv, auth)
	grpcusrinfo.Register(grpcsrv, usrInfo)
	grpcfollow.Register(grpcsrv, followProvider)

	return &App{
		log:     log,
		gRPCSrv: grpcsrv,
		port:    port,
	}
}

// logInterceptor wraps my logger to logging.Logger type
func logInterceptor(log *slog.Logger) logging.Logger {
	return logging.LoggerFunc(
		func(ctx context.Context, level logging.Level, msg string, fields ...any) {
			log.Log(ctx, slog.Level(level), msg, fields...)
		},
	)
}

// MustRun is wrapper of Run function which panics when error occurred
func (a *App) MustRun() {
	if err := a.Run(); err != nil {
		panic("failed to run application" + err.Error())
	}
}

// Run runs application
func (a *App) Run() error {
	const op = "grpcapp.Run"
	log := a.log.With(slog.String("op", op))
	log.Info("starting gRPC application")

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", a.port))
	if err != nil {
		log.Error(
			"failed to listen addr",
			sl.Err(err),
			slog.Int("port", a.port))
		return fmt.Errorf("%s: %w", op, err)
	}

	log.Info("starting to serve", slog.String("address", lis.Addr().String()))
	if err = a.gRPCSrv.Serve(lis); err != nil {
		log.Error("failed to serve socket", sl.Err(err))
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// Stop is graceful shutdown for application
func (a *App) Stop() {
	const op = "grpcapp.Stop"

	a.log.Info("stopping application", slog.String("op", op))

	a.gRPCSrv.GracefulStop()
}
