package main

import (
	"context"
	"fmt"
	"github.com/chempik1234/super-danis-library-golang/pkg/server"
	"github.com/chempik1234/super-danis-library-golang/pkg/server/grpcserver"
	"github.com/wb-go/wbf/zlog"
	"google.golang.org/grpc"
	"log"
	"net"
	"room_service/internal/config"
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
	zlog.InitConsole()
	err = zlog.SetLevel(cfg.Log.LogLevel)
	if err != nil {
		log.Fatal(fmt.Errorf("error setting log level to '%s': %w", cfg.Log.LogLevel, err))
	}
	zlog.Logger.Info().Msg("logger console init")
	//endregion

	// TODO: replace with gRPC
	grpcServer := grpc.NewServer()
	appServer := server.NewGracefulServer[*net.Listener](
		grpcserver.NewGracefulServerImplementationGRPC(grpcServer))

	//region run
	ctx, stopCtx := context.WithCancel(context.Background())
	defer stopCtx()

	zlog.Logger.Info().Int("grpc_port", cfg.Service.GRPCPort).Msg("server starting :grpc_port")
	err = appServer.GracefulRun(ctx, cfg.Service.GRPCPort)
	//endregion

	//region shutdown
	if err != nil {
		zlog.Logger.Error().Msg(fmt.Errorf("http server error: %w", err).Error())
	}

	zlog.Logger.Info().Msg("server gracefully stopped")

	stopCtx()
	zlog.Logger.Info().Msg("background operations gracefully stopped")
	fmt.Println("finish")
	//endregion
}
