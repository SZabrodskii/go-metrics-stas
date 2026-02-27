package agent

import (
	"context"
	"time"

	"github.com/SZabrodskii/go-metrics-stas/internal/model"
	pb "github.com/SZabrodskii/go-metrics-stas/internal/proto/metrics"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// grpcMetricsClient отправляет метрики на сервер по gRPC.
type grpcMetricsClient struct {
	conn    *grpc.ClientConn
	client  pb.MetricsClient
	localIP string
}

func newGRPCMetricsClient(address string, localIP string) (*grpcMetricsClient, error) {
	conn, err := grpc.NewClient(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}
	return &grpcMetricsClient{
		conn:    conn,
		client:  pb.NewMetricsClient(conn),
		localIP: localIP,
	}, nil
}

// SendBatch отправляет батч метрик через gRPC.
func (c *grpcMetricsClient) SendBatch(metrics []model.Metrics) error {
	pbMetrics := make([]*pb.Metric, 0, len(metrics))
	for _, m := range metrics {
		pm := &pb.Metric{Id: m.ID}
		switch m.MType {
		case model.Gauge:
			pm.Type = pb.Metric_GAUGE
			if m.Value != nil {
				pm.Value = *m.Value
			}
		case model.Counter:
			pm.Type = pb.Metric_COUNTER
			if m.Delta != nil {
				pm.Delta = *m.Delta
			}
		}
		pbMetrics = append(pbMetrics, pm)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if c.localIP != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "x-real-ip", c.localIP)
	}

	_, err := c.client.UpdateMetrics(ctx, &pb.UpdateMetricsRequest{Metrics: pbMetrics})
	return err
}

// Close закрывает gRPC соединение.
func (c *grpcMetricsClient) Close() error {
	return c.conn.Close()
}
