package interceptors

import (
	"context"
	"github.com/chempik1234/super-danis-library-golang/v2/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"time"
)

func AddLogMiddleware(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	ctx, _ = logger.New(ctx)
	ctx = context.WithValue(ctx, logger.KeyForRequestID, uuid.New().String())
	logger.GetLoggerFromCtx(ctx).Info(ctx, "gRPC request",
		zap.String("method", info.FullMethod),
		zap.Time("request time", time.Now()),
	)
	reply, err := handler(ctx, req)
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Warn(ctx, "gRPC hanler returned an error", zap.Error(err))
	}
	return reply, err
}

// AddLogMiddlewareStream logs gRPC stream requests
func AddLogMiddlewareStream(
	srv interface{},
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	ctx := ss.Context()
	ctx, _ = logger.New(ctx)
	ctx = context.WithValue(ctx, logger.KeyForRequestID, uuid.New().String())

	logger.GetLoggerFromCtx(ctx).Info(ctx, "gRPC stream request",
		zap.String("method", info.FullMethod),
		zap.Bool("is_client_stream", info.IsClientStream),
		zap.Bool("is_server_stream", info.IsServerStream),
		zap.Time("request_time", time.Now()),
	)

	err := handler(srv, ss)
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Warn(ctx, "gRPC stream handler returned an error", zap.Error(err))
	}
	return err
}
