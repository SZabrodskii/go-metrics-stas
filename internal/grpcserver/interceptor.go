package grpcserver

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TrustedSubnetInterceptor(cidr string) (grpc.UnaryServerInterceptor, error) {
	if cidr == "" {
		return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
			return handler(ctx, req)
		}, nil
	}

	_, subnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("invalid trusted_subnet CIDR: %w", err)
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.PermissionDenied, "missing metadata")
		}

		vals := md.Get("x-real-ip")
		if len(vals) == 0 || vals[0] == "" {
			return nil, status.Error(codes.PermissionDenied, "missing X-Real-IP")
		}

		ip := net.ParseIP(vals[0])
		if ip == nil {
			return nil, status.Error(codes.PermissionDenied, "invalid X-Real-IP")
		}

		if !subnet.Contains(ip) {
			return nil, status.Error(codes.PermissionDenied, "forbidden")
		}

		return handler(ctx, req)
	}, nil
}
