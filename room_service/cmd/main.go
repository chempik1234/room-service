package main

import (
	"context"
	"fmt"
	"github.com/chempik1234/room-service/internal/config"
	"github.com/chempik1234/room-service/internal/repositories/commandcache"
	mongodb2 "github.com/chempik1234/room-service/internal/repositories/room/mongodb"
	"github.com/chempik1234/room-service/internal/service/roomservice"
	"github.com/chempik1234/room-service/pkg/api/room_service"
	"github.com/chempik1234/room-service/pkg/transport/grpc/interceptors"
	"github.com/chempik1234/super-danis-library-golang/v2/pkg/logger"
	"github.com/chempik1234/super-danis-library-golang/v2/pkg/mongodb"
	"github.com/chempik1234/super-danis-library-golang/v2/pkg/redis"
	"github.com/chempik1234/super-danis-library-golang/v2/pkg/server"
	"github.com/chempik1234/super-danis-library-golang/v2/pkg/server/grpcserver"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
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

	//region mongodb
	mongoClient, err := mongodb.New(ctx, cfg.MongoDB)
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Error(ctx, "error creating mongodb client", zap.Error(err))
		return
	}
	defer mongodb.DeferDisconnect(ctx, mongoClient)
	logger.GetLoggerFromCtx(ctx).Info(ctx, "mongodb client created")
	//endregion

	//region redis
	redisClient, err := redis.New(ctx, cfg.Redis)
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Error(ctx, "error creating redis client", zap.Error(err))
		return
	}
	logger.GetLoggerFromCtx(ctx).Info(ctx, "redis client created")
	//endregion

	gRPCRetryStrategy := cfg.Service.RetryStrategy.ToStrategy()

	//region service
	var readConcern *readconcern.ReadConcern
	switch cfg.MongoDBRoomsRepo.ReadConcern {
	case "available":
		readConcern = readconcern.Available()
	case "local":
		readConcern = readconcern.Local()
	case "majority":
		readConcern = readconcern.Majority()
	case "linearizable":
		readConcern = readconcern.Linearizable()
	case "snapshot":
		readConcern = readconcern.Snapshot()
	default:
		panic(fmt.Errorf("unknown read concern: '%s' (Use one of these: 'available', 'local', 'majority', 'linearizable', 'snapshot')", cfg.MongoDBRoomsRepo.ReadConcern))
	}

	var writeConcern *writeconcern.WriteConcern
	switch cfg.MongoDBRoomsRepo.WriteConcern {
	case "w: 0":
		logger.GetLoggerFromCtx(ctx).Info(ctx, "mongo write_concern is set to w: 0")
		writeConcern = writeconcern.Unacknowledged()
		break
	case "w: 1":
		logger.GetLoggerFromCtx(ctx).Info(ctx, "mongo write_concern is set to w: 1")
		writeConcern = writeconcern.W1()
		break
	case "majority":
		logger.GetLoggerFromCtx(ctx).Info(ctx, "mongo write_concern is set to majority")
		writeConcern = writeconcern.Majority()
		break
	case "":
		logger.GetLoggerFromCtx(ctx).Info(ctx, "mongo write_concern is set to default (empty)")
		// Use default, writeConcern stays nil
		break
	default:
		logger.GetLoggerFromCtx(ctx).Info(ctx, "w concern is custom", zap.String("write_concern", cfg.MongoDBRoomsRepo.WriteConcern))
		writeConcern = writeconcern.Custom(cfg.MongoDBRoomsRepo.WriteConcern)
		break
	}

	// uses retry inside mongodb driver
	roomServiceServer := roomservice.NewRoomService(
		mongodb2.NewMongoDBRepository(mongoClient, mongodb2.MongoRepoParams{
			Database:       cfg.MongoDBRoomsRepo.Database,
			RoomCollection: cfg.MongoDBRoomsRepo.RoomsCollection,
			WriteConcern:   writeConcern,
			ReadConcern:    readConcern,
		}),
		commandcache.NewRedisCommandCache(redisClient, cfg.Redis.TTLSeconds*1000),
		gRPCRetryStrategy,
	)
	//endregion

	// Build interceptor chain based on config
	if cfg.Service.UseAuth {
		interceptors.SetAPIKey(cfg.Service.ApiKey)
	}
	unaryInterceptors, streamInterceptors := interceptors.MiddlewaresForService(cfg.Service.UseAuth)

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(unaryInterceptors...),
		grpc.ChainStreamInterceptor(streamInterceptors...),
	)
	room_service.RegisterRoomServiceServer(grpcServer, roomServiceServer)
	appServer := server.NewGracefulServer[*net.Listener](
		grpcserver.NewGracefulServerImplementationGRPC(grpcServer))

	//region run
	ctx, stopCtx := context.WithCancel(ctx)
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
