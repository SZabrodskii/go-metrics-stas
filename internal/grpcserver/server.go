package grpcserver

import (
	"context"

	"github.com/SZabrodskii/go-metrics-stas/internal/model"
	pb "github.com/SZabrodskii/go-metrics-stas/internal/proto/metrics"
	"github.com/SZabrodskii/go-metrics-stas/internal/service"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MetricsGRPCServer struct {
	pb.UnimplementedMetricsServer
	svc    service.MetricsService
	logger *zap.Logger
}

func NewMetricsGRPCServer(svc service.MetricsService, logger *zap.Logger) *MetricsGRPCServer {
	return &MetricsGRPCServer{svc: svc, logger: logger}
}

func (s *MetricsGRPCServer) UpdateMetrics(ctx context.Context, req *pb.UpdateMetricsRequest) (*pb.UpdateMetricsResponse, error) {
	batch := make([]model.Metrics, 0, len(req.GetMetrics()))
	for _, m := range req.GetMetrics() {
		mm := model.Metrics{
			ID: m.GetId(),
		}
		switch m.GetType() {
		case pb.Metric_GAUGE:
			mm.MType = model.Gauge
			v := m.GetValue()
			mm.Value = &v
		case pb.Metric_COUNTER:
			mm.MType = model.Counter
			d := m.GetDelta()
			mm.Delta = &d
		default:
			return nil, status.Errorf(codes.InvalidArgument, "unknown metric type")
		}
		batch = append(batch, mm)
	}

	if err := s.svc.UpdateBatch(batch); err != nil {
		s.logger.Error("gRPC UpdateMetrics failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "update batch: %v", err)
	}

	return &pb.UpdateMetricsResponse{}, nil
}
