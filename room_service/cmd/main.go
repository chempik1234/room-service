package main

import (
	"context"
	"fmt"
	"github.com/chempik1234/room-service/internal/config"
	"github.com/chempik1234/room-service/internal/repositories/commandcache"
	"github.com/chempik1234/room-service/internal/repositories/room"
	"github.com/chempik1234/room-service/internal/service/roomservice"
	"github.com/chempik1234/room-service/pkg/api/room_service"
	"github.com/chempik1234/room-service/pkg/transport/grpc/interceptors"
	"github.com/chempik1234/super-danis-library-golang/pkg/logger"
	"github.com/chempik1234/super-danis-library-golang/pkg/server"
	"github.com/chempik1234/super-danis-library-golang/pkg/server/grpcserver"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"log"
	"net"
)

func main() {
	//region load config from env
	var cfg, err = config.TryRead()
	if err != nil {
		log.Fatal(fmt.Errorf("error loading config: %w", err))
	}
	fmt.Println("config read")
	//endregion

	//region logger
	ctx, _ := logger.New(context.Background())

	logger.GetLoggerFromCtx(ctx).Info(ctx, "logger init")
	//endregion

	serviceRetryStrategy := cfg.RetryStrategy.ToStrategy()

	// TODO: roomRepo := commandIDCacheRepo := roomRetryStrategy :=
	roomServiceServer := roomservice.NewRoomService(room.NewMongoDBRepository(), commandcache.NewRedisCommandCache(), serviceRetryStrategy)

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(interceptors.AddLogMiddleware))
	room_service.RegisterRoomServiceServer(grpcServer, roomServiceServer)
	appServer := server.NewGracefulServer[*net.Listener](
		grpcserver.NewGracefulServerImplementationGRPC(grpcServer))

	//region run
	ctx, stopCtx := context.WithCancel(context.Background())
	defer stopCtx()

	logger.GetLoggerFromCtx(ctx).Info(ctx, "server starting :grpc_port", zap.Int("grpc_port", cfg.Service.GRPCPort))
	err = appServer.GracefulRun(ctx, cfg.Service.GRPCPort)
	//endregion

	//region shutdown
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Error(ctx, fmt.Errorf("http server error: %w", err).Error())
	}

	logger.GetLoggerFromCtx(ctx).Info(ctx, "server gracefully shutdown")

	stopCtx()
	logger.GetLoggerFromCtx(ctx).Info(ctx, "background operations gracefully shutdown")
	fmt.Println("finish")
	//endregion
}
