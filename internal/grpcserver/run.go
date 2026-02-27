package grpcserver

import (
	"context"
	"net"

	"github.com/SZabrodskii/go-metrics-stas/internal/config"
	pb "github.com/SZabrodskii/go-metrics-stas/internal/proto/metrics"
	"github.com/SZabrodskii/go-metrics-stas/internal/service"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// NewGRPCServer создаёт gRPC сервер с интерцептором проверки доверенной подсети.
func NewGRPCServer(cfg *config.ServerConfig, svc service.MetricsService, logger *zap.Logger) (*grpc.Server, error) {
	interceptor, err := TrustedSubnetInterceptor(cfg.TrustedSubnet)
	if err != nil {
		return nil, err
	}

	srv := grpc.NewServer(grpc.UnaryInterceptor(interceptor))
	pb.RegisterMetricsServer(srv, NewMetricsGRPCServer(svc, logger))
	return srv, nil
}

// RunGRPCServer запускает gRPC сервер в рамках fx lifecycle.
// Если GRPCAddress не задан — ничего не делает.
func RunGRPCServer(lc fx.Lifecycle, srv *grpc.Server, cfg *config.ServerConfig, logger *zap.Logger) {
	if cfg.GRPCAddress == "" {
		logger.Info("gRPC address not configured, skipping gRPC server")
		return
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			lis, err := net.Listen("tcp", cfg.GRPCAddress)
			if err != nil {
				return err
			}
			logger.Info("Starting gRPC server", zap.String("address", cfg.GRPCAddress))
			go func() {
				if err := srv.Serve(lis); err != nil {
					logger.Error("gRPC server error", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping gRPC server")
			srv.GracefulStop()
			return nil
		},
	})
}
