package happytest

import (
	"context"
	"testing"
	"time"
)

// TestStreamCreateRoom tests creating a room via stream
func TestStreamCreateRoom(t *testing.T) {
	client, err := NewTestClient(serviceAddr)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	stream, err := client.OpenStream(ctx)
	if err != nil {
		t.Fatalf("Failed to open stream: %v", err)
	}

	roomID, err := stream.CreateRoomStream(ctx, map[string]string{
		"max_users": "10",
		"game_type": "chess",
	})
	if err != nil {
		t.Fatalf("CreateRoomStream failed: %v", err)
	}

	if roomID == "" {
		t.Fatal("Expected non-empty room ID")
	}

	t.Logf("Created room via stream: %s", roomID)

	// Request snapshot to verify room exists
	if err := stream.RefreshRoomStream(roomID); err != nil {
		t.Fatalf("RefreshRoomStream failed: %v", err)
	}

	event, err := stream.Recv()
	if err != nil {
		t.Fatalf("Failed to receive snapshot event: %v", err)
	}

	snapshot := event.GetFullRoom()
	if snapshot == nil {
		t.Fatal("Expected FullRoom snapshot, got nil")
	}

	if snapshot.RoomId != roomID {
		t.Errorf("Expected room ID %s, got %s", roomID, snapshot.RoomId)
	}

	maxUsers := snapshot.RoomOptions["max_users"]
	if maxUsers != "10" {
		t.Errorf("Expected max_users=10, got %s", maxUsers)
	}
}

// TestStreamJoinLeaveRoom tests join and leave via stream with event listening
func TestStreamJoinLeaveRoom(t *testing.T) {
	client, err := NewTestClient(serviceAddr)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	stream, err := client.OpenStream(ctx)
	if err != nil {
		t.Fatalf("Failed to open stream: %v", err)
	}

	// Create a room
	roomID, err := stream.CreateRoomStream(ctx, nil)
	if err != nil {
		t.Fatalf("CreateRoomStream failed: %v", err)
	}
	t.Logf("Created room: %s", roomID)

	// User 1 joins
	if err := stream.JoinRoomStream(roomID, "user1", "Alice"); err != nil {
		t.Fatalf("JoinRoomStream failed for user1: %v", err)
	}

	// Receive the join event
	event, err := stream.Recv()
	if err != nil {
		t.Fatalf("Failed to receive join event: %v", err)
	}

	joined := event.GetJoinedRoom()
	if joined == nil {
		t.Fatal("Expected JoinedRoom event, got nil")
	}
	if joined.UserFull.Id != "user1" {
		t.Errorf("Expected user1, got %s", joined.UserFull.Id)
	}

	// User 2 joins
	if err := stream.JoinRoomStream(roomID, "user2", "Bob"); err != nil {
		t.Fatalf("JoinRoomStream failed for user2: %v", err)
	}

	event, err = stream.Recv()
	if err != nil {
		t.Fatalf("Failed to receive second join event: %v", err)
	}

	joined = event.GetJoinedRoom()
	if joined == nil || joined.UserFull.Id != "user2" {
		t.Errorf("Expected user2 joined event")
	}

	// Request snapshot to verify both users
	if err := stream.RefreshRoomStream(roomID); err != nil {
		t.Fatalf("RefreshRoomStream failed: %v", err)
	}

	event, err = stream.Recv()
	if err != nil {
		t.Fatalf("Failed to receive snapshot: %v", err)
	}

	snapshot := event.GetFullRoom()
	if len(snapshot.Users) != 2 {
		t.Errorf("Expected 2 users in snapshot, got %d", len(snapshot.Users))
	}

	// User 1 leaves
	if err := stream.LeaveRoomStream(roomID, "user1", "user1"); err != nil {
		t.Fatalf("LeaveRoomStream failed: %v", err)
	}

	event, err = stream.Recv()
	if err != nil {
		t.Fatalf("Failed to receive leave event: %v", err)
	}

	left := event.GetLeftRoom()
	if left == nil {
		t.Fatal("Expected LeftRoom event, got nil")
	}
	if left.KickedUserId != "user1" {
		t.Errorf("Expected user1 to leave, got %s", left.KickedUserId)
	}

	// Request final snapshot
	if err := stream.RefreshRoomStream(roomID); err != nil {
		t.Fatalf("RefreshRoomStream failed: %v", err)
	}

	event, err = stream.Recv()
	if err != nil {
		t.Fatalf("Failed to receive final snapshot: %v", err)
	}

	snapshot = event.GetFullRoom()
	if len(snapshot.Users) != 1 {
		t.Errorf("Expected 1 user after leave, got %d", len(snapshot.Users))
	}

	if snapshot.Users[0].Id != "user2" {
		t.Errorf("Expected user2, got %s", snapshot.Users[0].Id)
	}
}

// TestStreamSetDataStream tests setting data via stream and receiving events
func TestStreamSetDataStream(t *testing.T) {
	client, err := NewTestClient(serviceAddr)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	stream, err := client.OpenStream(ctx)
	if err != nil {
		t.Fatalf("Failed to open stream: %v", err)
	}

	// Create a room
	roomID, err := stream.CreateRoomStream(ctx, nil)
	if err != nil {
		t.Fatalf("CreateRoomStream failed: %v", err)
	}

	// Set some data
	if err := stream.SetDataStream(roomID, "game_state", StringValue("in_progress")); err != nil {
		t.Fatalf("SetDataStream failed: %v", err)
	}

	// Receive the data edited event
	event, err := stream.Recv()
	if err != nil {
		t.Fatalf("Failed to receive data edited event: %v", err)
	}

	dataEdited := event.GetDataEdited()
	if dataEdited == nil {
		t.Fatal("Expected DataEdited event, got nil")
	}
	if dataEdited.DataId != "game_state" {
		t.Errorf("Expected data_id=game_state, got %s", dataEdited.DataId)
	}
	if dataEdited.DataValue.GetStringValue() != "in_progress" {
		t.Errorf("Expected game_state='in_progress', got '%s'", dataEdited.DataValue.GetStringValue())
	}

	// Set more data
	if err := stream.SetDataStream(roomID, "turn_counter", IntValue(5)); err != nil {
		t.Fatalf("SetDataStream failed: %v", err)
	}

	event, err = stream.Recv()
	if err != nil {
		t.Fatalf("Failed to receive second data edited event: %v", err)
	}

	dataEdited = event.GetDataEdited()
	if dataEdited == nil || dataEdited.DataId != "turn_counter" {
		t.Errorf("Expected turn_counter data edited event")
	}

	// Verify via snapshot
	if err := stream.RefreshRoomStream(roomID); err != nil {
		t.Fatalf("RefreshRoomStream failed: %v", err)
	}

	event, err = stream.Recv()
	if err != nil {
		t.Fatalf("Failed to receive snapshot: %v", err)
	}

	snapshot := event.GetFullRoom()
	if snapshot == nil || snapshot.Room == nil {
		t.Fatal("Expected valid room snapshot")
	}

	if snapshot.Room.Values["game_state"].GetStringValue() != "in_progress" {
		t.Errorf("Expected game_state='in_progress' in snapshot")
	}

	if snapshot.Room.Values["turn_counter"].GetIntValue() != 5 {
		t.Errorf("Expected turn_counter=5 in snapshot")
	}
}

// TestStreamSetOwner tests setting the owner of a room via stream
func TestStreamSetOwner(t *testing.T) {
	client, err := NewTestClient(serviceAddr)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	stream, err := client.OpenStream(ctx)
	if err != nil {
		t.Fatalf("Failed to open stream: %v", err)
	}

	// Create a room
	roomID, err := stream.CreateRoomStream(ctx, nil)
	if err != nil {
		t.Fatalf("CreateRoomStream failed: %v", err)
	}
	_, _ = stream.Recv() // Receive room created event

	t.Logf("Created room: %s", roomID)

	// Set owner to a new user
	if err := stream.SetOwnerStream(roomID, "owner-user-1"); err != nil {
		t.Fatalf("SetOwnerStream failed: %v", err)
	}

	// Receive the owner changed event
	event, err := stream.Recv()
	if err != nil {
		t.Fatalf("Failed to receive owner changed event: %v", err)
	}

	ownerChanged := event.GetOwnerChanged()
	if ownerChanged == nil {
		t.Fatal("Expected OwnerChanged event, got nil")
	}
	if ownerChanged.NewOwnerId != "owner-user-1" {
		t.Errorf("Expected new owner ID 'owner-user-1', got '%s'", ownerChanged.NewOwnerId)
	}
	if !ownerChanged.OwnerHasChanged {
		t.Error("Expected OwnerHasChanged to be true")
	}

	t.Logf("Owner changed to: %s (changed: %v)", ownerChanged.NewOwnerId, ownerChanged.OwnerHasChanged)

	// Set owner to the same user - should indicate no change
	if err := stream.SetOwnerStream(roomID, "owner-user-1"); err != nil {
		t.Fatalf("SetOwnerStream (second call) failed: %v", err)
	}

	event, err = stream.Recv()
	if err != nil {
		t.Fatalf("Failed to receive second owner changed event: %v", err)
	}

	ownerChanged = event.GetOwnerChanged()
	if ownerChanged.OwnerHasChanged {
		t.Error("Expected OwnerHasChanged to be false when setting to same owner")
	}

	// Set owner to a different user
	if err := stream.SetOwnerStream(roomID, "owner-user-2"); err != nil {
		t.Fatalf("SetOwnerStream (third call) failed: %v", err)
	}

	event, err = stream.Recv()
	if err != nil {
		t.Fatalf("Failed to receive third owner changed event: %v", err)
	}

	ownerChanged = event.GetOwnerChanged()
	if ownerChanged.NewOwnerId != "owner-user-2" {
		t.Errorf("Expected new owner ID 'owner-user-2', got '%s'", ownerChanged.NewOwnerId)
	}
	if !ownerChanged.OwnerHasChanged {
		t.Error("Expected OwnerHasChanged to be true when changing to different owner")
	}

	t.Logf("Owner changed to: %s (changed: %v)", ownerChanged.NewOwnerId, ownerChanged.OwnerHasChanged)
}

// TestStreamMultipleOperations tests multiple operations in a single stream
func TestStreamMultipleOperations(t *testing.T) {
	client, err := NewTestClient(serviceAddr)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	stream, err := client.OpenStream(ctx)
	if err != nil {
		t.Fatalf("Failed to open stream: %v", err)
	}

	// Create room
	roomID, err := stream.CreateRoomStream(ctx, map[string]string{"name": "Test Room"})
	if err != nil {
		t.Fatalf("CreateRoomStream failed: %v", err)
	}

	// Receive room created event
	event, err := stream.Recv()
	if err != nil {
		t.Fatalf("Failed to receive room created event: %v", err)
	}
	if event.GetRoomCreated() == nil {
		t.Fatal("Expected RoomCreated event")
	}

	// Join users
	if err := stream.JoinRoomStream(roomID, "alice", "Alice"); err != nil {
		t.Fatalf("JoinRoomStream for alice failed: %v", err)
	}
	if err := stream.JoinRoomStream(roomID, "bob", "Bob"); err != nil {
		t.Fatalf("JoinRoomStream for bob failed: %v", err)
	}

	// Receive join events
	for i := 0; i < 2; i++ {
		event, err = stream.Recv()
		if err != nil {
			t.Fatalf("Failed to receive join event %d: %v", i+1, err)
		}
		if event.GetJoinedRoom() == nil {
			t.Errorf("Expected JoinedRoom event %d", i+1)
		}
	}

	// Set data
	if err := stream.SetDataStream(roomID, "score", IntValue(100)); err != nil {
		t.Fatalf("SetDataStream failed: %v", err)
	}

	event, err = stream.Recv()
	if err != nil {
		t.Fatalf("Failed to receive data edited event: %v", err)
	}
	if event.GetDataEdited() == nil {
		t.Fatal("Expected DataEdited event")
	}

	// Request final snapshot
	if err := stream.RefreshRoomStream(roomID); err != nil {
		t.Fatalf("RefreshRoomStream failed: %v", err)
	}

	event, err = stream.Recv()
	if err != nil {
		t.Fatalf("Failed to receive snapshot: %v", err)
	}

	snapshot := event.GetFullRoom()
	if len(snapshot.Users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(snapshot.Users))
	}
	if snapshot.Room.Values["score"].GetIntValue() != 100 {
		t.Errorf("Expected score=100, got %d", snapshot.Room.Values["score"].GetIntValue())
	}
}

// TestStreamDeleteRoom tests deleting a room via stream
func TestStreamDeleteRoom(t *testing.T) {
	client, err := NewTestClient(serviceAddr)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	stream, err := client.OpenStream(ctx)
	if err != nil {
		t.Fatalf("Failed to open stream: %v", err)
	}

	// Create a room
	roomID, err := stream.CreateRoomStream(ctx, nil)
	if err != nil {
		t.Fatalf("CreateRoomStream failed: %v", err)
	}

	// Receive room created event
	_, err = stream.Recv()
	if err != nil {
		t.Fatalf("Failed to receive room created event: %v", err)
	}

	// Delete the room
	if err := stream.DeleteRoomStream(roomID); err != nil {
		t.Fatalf("DeleteRoomStream failed: %v", err)
	}

	// Receive delete event
	event, err := stream.Recv()
	if err != nil {
		t.Fatalf("Failed to receive delete event: %v", err)
	}

	deleted := event.GetRoomDeleted()
	if deleted == nil {
		t.Fatal("Expected RoomDeleted event, got nil")
	}
	if deleted.DeletedRoomId != roomID {
		t.Errorf("Expected deleted room ID %s, got %s", roomID, deleted.DeletedRoomId)
	}
}

// TestStreamTwoClients tests two concurrent streams interacting
func TestStreamTwoClients(t *testing.T) {
	ctx := context.Background()

	// Client 1 creates the room
	client1, err := NewTestClient(serviceAddr)
	if err != nil {
		t.Fatalf("Failed to create client1: %v", err)
	}
	defer client1.Close()

	stream1, err := client1.OpenStream(ctx)
	if err != nil {
		t.Fatalf("Failed to open stream1: %v", err)
	}

	roomID, err := stream1.CreateRoomStream(ctx, nil)
	if err != nil {
		t.Fatalf("CreateRoomStream on client1 failed: %v", err)
	}
	_, _ = stream1.Recv() // Receive room created event

	t.Logf("Client1 created room: %s", roomID)

	// Client 2 joins the room
	client2, err := NewTestClient(serviceAddr)
	if err != nil {
		t.Fatalf("Failed to create client2: %v", err)
	}
	defer client2.Close()

	stream2, err := client2.OpenStream(ctx)
	if err != nil {
		t.Fatalf("Failed to open stream2: %v", err)
	}

	// Client 2 joins
	if err := stream2.JoinRoomStream(roomID, "client2_user", "Client2User"); err != nil {
		t.Fatalf("JoinRoomStream on client2 failed: %v", err)
	}

	// Both clients should receive the join event
	done := make(chan bool, 2)

	go func() {
		event, err := stream1.Recv()
		if err != nil {
			t.Errorf("Client1 failed to receive event: %v", err)
		}
		if event.GetJoinedRoom() == nil {
			t.Error("Client1: Expected JoinedRoom event")
		}
		t.Log("Client1 received join event from client2")
		done <- true
	}()

	go func() {
		event, err := stream2.Recv()
		if err != nil {
			t.Errorf("Client2 failed to receive event: %v", err)
		}
		if event.GetJoinedRoom() == nil {
			t.Error("Client2: Expected JoinedRoom event")
		}
		t.Log("Client2 received own join event")
		done <- true
	}()

	// Wait for both with timeout
	timeout := time.After(5 * time.Second)
	for i := 0; i < 2; i++ {
		select {
		case <-done:
			// OK
		case <-timeout:
			t.Fatal("Timeout waiting for events")
		}
	}

	// Client 1 sets data
	if err := stream1.SetDataStream(roomID, "shared_data", StringValue("hello")); err != nil {
		t.Fatalf("SetDataStream on client1 failed: %v", err)
	}

	// Both clients should receive the data edited event
	done = make(chan bool, 2)

	go func() {
		event, err := stream1.Recv()
		if err != nil {
			t.Errorf("Client1 failed to receive data event: %v", err)
		}
		if event.GetDataEdited() == nil {
			t.Error("Client1: Expected DataEdited event")
		}
		t.Log("Client1 received data edited event")
		done <- true
	}()

	go func() {
		event, err := stream2.Recv()
		if err != nil {
			t.Errorf("Client2 failed to receive data event: %v", err)
		}
		if event.GetDataEdited() == nil {
			t.Error("Client2: Expected DataEdited event")
		}
		if event.GetDataEdited().DataValue.GetStringValue() != "hello" {
			t.Errorf("Client2: Expected data='hello', got '%s'", event.GetDataEdited().DataValue.GetStringValue())
		}
		t.Log("Client2 received data edited event from client1")
		done <- true
	}()

	for i := 0; i < 2; i++ {
		select {
		case <-done:
			// OK
		case <-timeout:
			t.Fatal("Timeout waiting for data events")
		}
	}
}

// TestStreamConcurrentDataUpdates tests rapid concurrent data updates
func TestStreamConcurrentDataUpdates(t *testing.T) {
	client, err := NewTestClient(serviceAddr)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	stream, err := client.OpenStream(ctx)
	if err != nil {
		t.Fatalf("Failed to open stream: %v", err)
	}

	roomID, err := stream.CreateRoomStream(ctx, nil)
	if err != nil {
		t.Fatalf("CreateRoomStream failed: %v", err)
	}
	_, _ = stream.Recv() // Receive room created event

	// Send multiple data updates
	numUpdates := 5
	for i := 0; i < numUpdates; i++ {
		dataID := "counter_" + string(rune('A'+i))
		value := IntValue(int64(i * 10))
		if err := stream.SetDataStream(roomID, dataID, value); err != nil {
			t.Fatalf("SetDataStream %d failed: %v", i, err)
		}
	}

	// Receive all events
	received := 0
	for received < numUpdates {
		event, err := stream.Recv()
		if err != nil {
			t.Fatalf("Failed to receive event %d: %v", received+1, err)
		}

		if event.GetDataEdited() != nil {
			received++
			t.Logf("Received data edited event: %s = %d",
				event.GetDataEdited().DataId,
				event.GetDataEdited().DataValue.GetIntValue())
		}
	}

	if received != numUpdates {
		t.Errorf("Expected %d data edited events, received %d", numUpdates, received)
	}

	// Verify final state via snapshot
	if err := stream.RefreshRoomStream(roomID); err != nil {
		t.Fatalf("RefreshRoomStream failed: %v", err)
	}

	event, err := stream.Recv()
	if err != nil {
		t.Fatalf("Failed to receive snapshot: %v", err)
	}

	snapshot := event.GetFullRoom()
	if len(snapshot.Room.Values) != numUpdates {
		t.Errorf("Expected %d values in snapshot, got %d", numUpdates, len(snapshot.Room.Values))
	}
}

// TestStreamErrorHandling tests error scenarios in streaming mode
func TestStreamErrorHandling(t *testing.T) {
	client, err := NewTestClient(serviceAddr)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	stream, err := client.OpenStream(ctx)
	if err != nil {
		t.Fatalf("Failed to open stream: %v", err)
	}

	// Try to join a non-existent room
	fakeRoomID := "nonexistent-room-12345"
	if err := stream.JoinRoomStream(fakeRoomID, "user1", "Alice"); err != nil {
		t.Fatalf("JoinRoomStream send failed: %v", err)
	}

	// Should receive an error event
	event, err := stream.Recv()
	if err != nil {
		t.Fatalf("Failed to receive error event: %v", err)
	}

	errorMsg := event.GetErrorMessage()
	if errorMsg == nil {
		t.Error("Expected ErrorMessage event, got nil")
	} else {
		t.Logf("Received expected error: %s", errorMsg.Error)
	}
}
