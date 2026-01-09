package main

import (
	"context"
	"fmt"
	"github.com/chempik1234/super-danis-library-golang/v2/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"happytest/pkg/api/room_service"
)

func main() {
	grpcClient, err := grpc.NewClient("localhost:50050", grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		panic(err)
	}

	ctx, _ := logger.New(context.Background())

	grpcClientRooms := room_service.NewRoomServiceClient(grpcClient)

	fmt.Println(ctx, grpcClientRooms)
	fmt.Println("hello world!")
}
