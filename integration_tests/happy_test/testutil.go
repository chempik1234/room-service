package happytest

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"happytest/pkg/api/room_service"
)

// TestClient wraps the gRPC client with test helpers
type TestClient struct {
	client room_service.RoomServiceClient
	conn   *grpc.ClientConn
}

// NewTestClient creates a new test client connection
func NewTestClient(addr string) (*TestClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
	}

	return &TestClient{
		client: room_service.NewRoomServiceClient(conn),
		conn:   conn,
	}, nil
}

// Close closes the connection
func (tc *TestClient) Close() error {
	return tc.conn.Close()
}

// CreateRoom creates a room and returns the room ID
func (tc *TestClient) CreateRoom(ctx context.Context, opts map[string]string) (string, error) {
	cmd := &room_service.Command{
		Timestamp: time.Now().UnixMicro(),
		UserId:    "test-user",
		Payload: &room_service.Command_CreateRoom{
			CreateRoom: &room_service.CreateRoomCommandBody{
				RoomOptions: opts,
			},
		},
	}

	resp, err := tc.client.SingleCommand(ctx, cmd)
	if err != nil {
		return "", fmt.Errorf("create room failed: %w", err)
	}

	log.Printf("DEBUG: CreateRoom response: %+v", resp)

	room := resp.GetRoomCreated()
	if room == nil {
		// Check if it's a delete response
		if deleted := resp.GetRoomDeleted(); deleted != nil {
			return "", fmt.Errorf("got RoomDeleted response instead: %s", deleted.DeletedRoomId)
		}
		return "", fmt.Errorf("expected RoomCreated response, got nil (response: %+v)", resp)
	}

	if room.GetRoomId() == "" {
		return "", fmt.Errorf("expected non-empty room ID")
	}

	log.Printf("DEBUG: Created room ID: %s", room.GetRoomId())
	return room.GetRoomId(), nil
}

// JoinRoom joins a user to a room
func (tc *TestClient) JoinRoom(ctx context.Context, roomID, userID, userName string) error {
	cmd := &room_service.Command{
		Timestamp: time.Now().UnixMicro(),
		UserId:    userID,
		RoomId:    &roomID,
		Payload: &room_service.Command_JoinRoom{
			JoinRoom: &room_service.JoinRoomCommandBody{
				UserFull: &room_service.User{
					Id:   userID,
					Name: userName,
				},
			},
		},
	}

	_, err := tc.client.SingleCommand(ctx, cmd)
	return err
}

// LeaveRoom makes a user leave a room
func (tc *TestClient) LeaveRoom(ctx context.Context, roomID, userID, kickedUserID string) error {
	cmd := &room_service.Command{
		Timestamp: time.Now().UnixMicro(),
		UserId:    userID,
		RoomId:    &roomID,
		Payload: &room_service.Command_LeaveRoom{
			LeaveRoom: &room_service.LeaveRoomCommandBody{
				KickedUserId: kickedUserID,
			},
		},
	}

	_, err := tc.client.SingleCommand(ctx, cmd)
	return err
}

// DeleteRoom deletes a room
func (tc *TestClient) DeleteRoom(ctx context.Context, roomID string) error {
	cmd := &room_service.Command{
		Timestamp: time.Now().UnixMicro(),
		UserId:    "test-user",
		RoomId:    &roomID,
		Payload: &room_service.Command_DeleteRoom{
			DeleteRoom: &room_service.DeleteRoomCommandBody{
				DeleteApprove: true,
			},
		},
	}

	_, err := tc.client.SingleCommand(ctx, cmd)
	return err
}

// SetData sets a data value in a room
func (tc *TestClient) SetData(ctx context.Context, roomID, dataID string, value *room_service.Value) error {
	cmd := &room_service.Command{
		Timestamp: time.Now().UnixMicro(),
		UserId:    "test-user",
		RoomId:    &roomID,
		Payload: &room_service.Command_AffectData{
			AffectData: &room_service.SetAppendDeleteDataCommandBody{
				DataId:      dataID,
				DataValue:   value,
				CommandMode: room_service.DateEditMode_SET,
			},
		},
	}

	_, err := tc.client.SingleCommand(ctx, cmd)
	return err
}

// GetRoomSnapshot fetches the current room state
func (tc *TestClient) GetRoomSnapshot(ctx context.Context, roomID string) (*room_service.FullRoomSnapshotEventBody, error) {
	cmd := &room_service.Command{
		Timestamp: time.Now().UnixMicro(),
		UserId:    "test-user",
		RoomId:    &roomID,
		Payload: &room_service.Command_RefreshRoom{
			RefreshRoom: &room_service.RefreshRoomCommandBody{
				RefreshRoom: true,
			},
		},
	}

	resp, err := tc.client.SingleCommand(ctx, cmd)
	if err != nil {
		return nil, err
	}

	return resp.GetFullRoom(), nil
}

// WaitForService waits for the service to be ready
func WaitForService(addr string, maxAttempts int) error {
	log.Printf("Waiting for service at %s...", addr)
	for i := 0; i < maxAttempts; i++ {
		_, err := NewTestClient(addr)
		if err == nil {
			log.Println("Service is ready!")
			return nil
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("service not ready after %d attempts", maxAttempts)
}

// StringValue creates a string Value
func StringValue(s string) *room_service.Value {
	return &room_service.Value{
		Value: &room_service.Value_StringValue{StringValue: s},
	}
}

// IntValue creates an int Value
func IntValue(i int64) *room_service.Value {
	return &room_service.Value{
		Value: &room_service.Value_IntValue{IntValue: i},
	}
}
