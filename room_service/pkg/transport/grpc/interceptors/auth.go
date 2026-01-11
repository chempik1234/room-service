package interceptors

import (
	"context"
	"errors"

	"github.com/chempik1234/super-danis-library-golang/v2/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	// APIKeyHeader is the metadata key for API key authentication
	APIKeyHeader = "x-api-key"
)

// apiKey is the configured API key for authentication
// Set by SetAPIKey before starting the server
var apiKey string = "apikey" // default for backwards compatibility

// SetAPIKey sets the API key for authentication
func SetAPIKey(key string) {
	apiKey = key
}

// APIKeyAuth validates API key for unary RPCs
func APIKeyAuth(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	clientKey, err := extractAPIKey(ctx)
	if err != nil || clientKey != apiKey {
		logger.GetOrCreateLoggerFromCtx(ctx).Warn(ctx, "API key validation failed",
			zap.String("method", info.FullMethod),
			zap.Error(err),
		)
		return nil, status.Errorf(codes.Unauthenticated, "invalid API key")
	}

	return handler(ctx, req)
}

// APIKeyAuthStream validates API key for streaming RPCs
func APIKeyAuthStream(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	ctx := ss.Context()

	clientKey, err := extractAPIKey(ctx)
	if err != nil || clientKey != apiKey {
		logger.GetOrCreateLoggerFromCtx(ctx).Warn(ctx, "API key validation failed (stream)",
			zap.String("method", info.FullMethod),
			zap.Error(err),
		)
		return status.Errorf(codes.Unauthenticated, "invalid API key")
	}

	return handler(srv, ss)
}

// extractAPIKey gets API key from gRPC metadata
func extractAPIKey(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", errors.New("no metadata")
	}

	keys := md.Get(APIKeyHeader)
	if len(keys) == 0 {
		return "", errors.New("API key not found")
	}

	return keys[0], nil
}
