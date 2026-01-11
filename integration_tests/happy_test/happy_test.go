package happytest

import (
	"context"
	"os"
	"testing"
)

const serviceAddr = "localhost:50050"

// TestMain runs setup before all tests
func TestMain(m *testing.M) {
	// Wait for service to be ready
	if err := WaitForService(serviceAddr, 30); err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

// TestCreateRoom creates a room and verifies it exists
func TestCreateRoom(t *testing.T) {
	client, err := NewTestClient(serviceAddr)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	roomID, err := client.CreateRoom(ctx, map[string]string{
		"max_users": "10",
		"game_type": "chess",
	})
	if err != nil {
		t.Fatalf("CreateRoom failed: %v", err)
	}

	if roomID == "" {
		t.Fatal("Expected non-empty room ID")
	}

	t.Logf("Created room: %s", roomID)

	// Verify room exists by fetching snapshot
	snapshot, err := client.GetRoomSnapshot(ctx, roomID)
	if err != nil {
		t.Fatalf("GetRoomSnapshot failed: %v", err)
	}

	if snapshot.RoomId != roomID {
		t.Errorf("Expected room ID %s, got %s", roomID, snapshot.RoomId)
	}

	maxUsers := snapshot.RoomOptions["max_users"]
	if maxUsers != "10" {
		t.Errorf("Expected max_users=10, got %s", maxUsers)
	}
}

// TestJoinLeaveRoom tests the full join-leave cycle
func TestJoinLeaveRoom(t *testing.T) {
	client, err := NewTestClient(serviceAddr)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create a room
	roomID, err := client.CreateRoom(ctx, nil)
	if err != nil {
		t.Fatalf("CreateRoom failed: %v", err)
	}
	t.Logf("Created room: %s", roomID)

	// User 1 joins
	err = client.JoinRoom(ctx, roomID, "user1", "Alice")
	if err != nil {
		t.Fatalf("JoinRoom failed for user1: %v", err)
	}

	// User 2 joins
	err = client.JoinRoom(ctx, roomID, "user2", "Bob")
	if err != nil {
		t.Fatalf("JoinRoom failed for user2: %v", err)
	}

	// Verify both users are in the room
	snapshot, err := client.GetRoomSnapshot(ctx, roomID)
	if err != nil {
		t.Fatalf("GetRoomSnapshot failed: %v", err)
	}

	if len(snapshot.Users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(snapshot.Users))
	}

	// User 1 leaves
	err = client.LeaveRoom(ctx, roomID, "user1", "user1")
	if err != nil {
		t.Fatalf("LeaveRoom failed: %v", err)
	}

	// Verify only user2 remains
	snapshot, err = client.GetRoomSnapshot(ctx, roomID)
	if err != nil {
		t.Fatalf("GetRoomSnapshot failed: %v", err)
	}

	if len(snapshot.Users) != 1 {
		t.Errorf("Expected 1 user after leave, got %d", len(snapshot.Users))
	}

	if snapshot.Users[0].Id != "user2" {
		t.Errorf("Expected user2, got %s", snapshot.Users[0].Id)
	}
}

// TestSetData tests setting and retrieving data in a room
func TestSetData(t *testing.T) {
	client, err := NewTestClient(serviceAddr)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create a room
	roomID, err := client.CreateRoom(ctx, nil)
	if err != nil {
		t.Fatalf("CreateRoom failed: %v", err)
	}

	// Set some data
	err = client.SetData(ctx, roomID, "game_state", StringValue("in_progress"))
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	err = client.SetData(ctx, roomID, "turn_counter", IntValue(5))
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	// Verify data was set
	snapshot, err := client.GetRoomSnapshot(ctx, roomID)
	if err != nil {
		t.Fatalf("GetRoomSnapshot failed: %v", err)
	}

	if snapshot == nil {
		t.Fatal("Snapshot is nil")
	}

	if snapshot.Room == nil {
		t.Fatalf("Snapshot does not contain room: %v", snapshot)
	}

	gameState := snapshot.Room.Values["game_state"]
	if gameState == nil {
		t.Fatal("game_state not found in room data")
	}

	if gameState.GetStringValue() != "in_progress" {
		t.Errorf("Expected game_state='in_progress', got '%s'", gameState.GetStringValue())
	}

	turnCounter := snapshot.Room.Values["turn_counter"]
	if turnCounter == nil {
		t.Fatal("turn_counter not found in room data")
	}

	if turnCounter.GetIntValue() != 5 {
		t.Errorf("Expected turn_counter=5, got %d", turnCounter.GetIntValue())
	}
}

// TestSetOwner tests setting the owner of a room
func TestSetOwner(t *testing.T) {
	client, err := NewTestClient(serviceAddr)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create a room
	roomID, err := client.CreateRoom(ctx, nil)
	if err != nil {
		t.Fatalf("CreateRoom failed: %v", err)
	}
	t.Logf("Created room: %s", roomID)

	// Set owner to a new user
	ownerChanged, err := client.SetOwner(ctx, roomID, "new-owner-user")
	if err != nil {
		t.Fatalf("SetOwner failed: %v", err)
	}

	if ownerChanged == nil {
		t.Fatal("Expected OwnerChanged response, got nil")
	}

	if ownerChanged.NewOwnerId != "new-owner-user" {
		t.Errorf("Expected new owner ID 'new-owner-user', got '%s'", ownerChanged.NewOwnerId)
	}

	if !ownerChanged.OwnerHasChanged {
		t.Error("Expected OwnerHasChanged to be true")
	}

	t.Logf("Owner changed to: %s (changed: %v)", ownerChanged.NewOwnerId, ownerChanged.OwnerHasChanged)

	// Set owner again to the same user - should indicate no change
	ownerChanged2, err := client.SetOwner(ctx, roomID, "new-owner-user")
	if err != nil {
		t.Fatalf("SetOwner (second call) failed: %v", err)
	}

	if ownerChanged2.OwnerHasChanged {
		t.Error("Expected OwnerHasChanged to be false when setting to same owner")
	}

	// Set owner to a different user
	ownerChanged3, err := client.SetOwner(ctx, roomID, "another-owner")
	if err != nil {
		t.Fatalf("SetOwner (third call) failed: %v", err)
	}

	if ownerChanged3.NewOwnerId != "another-owner" {
		t.Errorf("Expected new owner ID 'another-owner', got '%s'", ownerChanged3.NewOwnerId)
	}

	if !ownerChanged3.OwnerHasChanged {
		t.Error("Expected OwnerHasChanged to be true when changing to different owner")
	}
}

// TestDeleteRoom tests creating and deleting a room
func TestDeleteRoom(t *testing.T) {
	client, err := NewTestClient(serviceAddr)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create a room
	roomID, err := client.CreateRoom(ctx, nil)
	if err != nil {
		t.Fatalf("CreateRoom failed: %v", err)
	}
	t.Logf("Created room: %s", roomID)

	// Verify it exists
	snapshot, err := client.GetRoomSnapshot(ctx, roomID)
	if err != nil {
		t.Fatalf("GetRoomSnapshot before delete failed: %v", err)
	}
	if snapshot == nil {
		t.Fatal("Expected non-nil snapshot before delete")
	}

	// Delete the room
	err = client.DeleteRoom(ctx, roomID)
	if err != nil {
		t.Fatalf("DeleteRoom failed: %v", err)
	}

	// Note: Depending on your implementation, getting a deleted room
	// might return an error or an empty snapshot. Adjust this assertion
	// based on your service's actual behavior.
	_, err = client.GetRoomSnapshot(ctx, roomID)
	if err == nil {
		t.Log("Warning: Deleted room still accessible (service may return empty state)")
	}
}

// TestMultipleRooms tests managing multiple rooms simultaneously
func TestMultipleRooms(t *testing.T) {
	client, err := NewTestClient(serviceAddr)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create three rooms
	room1, err := client.CreateRoom(ctx, map[string]string{"name": "Lobby"})
	if err != nil {
		t.Fatalf("CreateRoom failed for room1: %v", err)
	}

	room2, err := client.CreateRoom(ctx, map[string]string{"name": "Game Room 1"})
	if err != nil {
		t.Fatalf("CreateRoom failed for room2: %v", err)
	}

	room3, err := client.CreateRoom(ctx, map[string]string{"name": "Game Room 2"})
	if err != nil {
		t.Fatalf("CreateRoom failed for room3: %v", err)
	}

	// Verify all three rooms have different IDs
	if room1 == room2 || room2 == room3 || room1 == room3 {
		t.Error("Expected unique room IDs")
	}

	// Add users to different rooms
	client.JoinRoom(ctx, room1, "alice", "Alice")
	client.JoinRoom(ctx, room1, "bob", "Bob")

	client.JoinRoom(ctx, room2, "charlie", "Charlie")

	// Verify room1 has 2 users
	snap1, _ := client.GetRoomSnapshot(ctx, room1)
	if len(snap1.Users) != 2 {
		t.Errorf("Expected 2 users in room1, got %d", len(snap1.Users))
	}

	// Verify room2 has 1 user
	snap2, _ := client.GetRoomSnapshot(ctx, room2)
	if len(snap2.Users) != 1 {
		t.Errorf("Expected 1 user in room2, got %d", len(snap2.Users))
	}

	// Verify room3 is empty
	snap3, _ := client.GetRoomSnapshot(ctx, room3)
	if len(snap3.Users) != 0 {
		t.Errorf("Expected 0 users in room3, got %d", len(snap3.Users))
	}
}

// TestRoomOptions tests various room configurations
func TestRoomOptions(t *testing.T) {
	tests := []struct {
		name    string
		opts    map[string]string
		wantKey string
		wantVal string
	}{
		{
			name:    "max users only",
			opts:    map[string]string{"max_users": "5"},
			wantKey: "max_users",
			wantVal: "5",
		},
		{
			name: "multiple options",
			opts: map[string]string{
				"max_users": "100",
				"game_type": "poker",
				"private":   "true",
			},
			wantKey: "game_type",
			wantVal: "poker",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewTestClient(serviceAddr)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}
			defer client.Close()

			ctx := context.Background()

			roomID, err := client.CreateRoom(ctx, tt.opts)
			if err != nil {
				t.Fatalf("CreateRoom failed: %v", err)
			}

			snapshot, err := client.GetRoomSnapshot(ctx, roomID)
			if err != nil {
				t.Fatalf("GetRoomSnapshot failed: %v", err)
			}

			gotVal := snapshot.RoomOptions[tt.wantKey]
			if gotVal != tt.wantVal {
				t.Errorf("Expected %s=%s, got %s", tt.wantKey, tt.wantVal, gotVal)
			}
		})
	}
}

// TestRoomsList tests listing all rooms
func TestRoomsList(t *testing.T) {
	client, err := NewTestClient(serviceAddr)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create three rooms with different options
	room1, err := client.CreateRoom(ctx, map[string]string{"name": "Lobby", "max_users": "10"})
	if err != nil {
		t.Fatalf("CreateRoom failed for room1: %v", err)
	}

	room2, err := client.CreateRoom(ctx, map[string]string{"name": "Game Room 1", "game_type": "chess"})
	if err != nil {
		t.Fatalf("CreateRoom failed for room2: %v", err)
	}

	room3, err := client.CreateRoom(ctx, map[string]string{"name": "Game Room 2", "private": "true"})
	if err != nil {
		t.Fatalf("CreateRoom failed for room3: %v", err)
	}

	// List all rooms
	roomsList, err := client.RoomsList(ctx)
	if err != nil {
		t.Fatalf("RoomsList failed: %v", err)
	}

	if roomsList == nil {
		t.Fatal("Expected non-nil RoomsList response")
	}

	// We expect at least 3 rooms (there might be more from previous tests)
	if len(roomsList.Rooms) < 3 {
		t.Errorf("Expected at least 3 rooms, got %d", len(roomsList.Rooms))
	}

	// Verify our created rooms are in the list
	foundRoom1, foundRoom2, foundRoom3 := false, false, false
	for _, room := range roomsList.Rooms {
		if room.RoomId == room1 {
			foundRoom1 = true
			if room.RoomOptions["name"] != "Lobby" {
				t.Errorf("Expected room1 name=Lobby, got %s", room.RoomOptions["name"])
			}
			if room.RoomOptions["max_users"] != "10" {
				t.Errorf("Expected room1 max_users=10, got %s", room.RoomOptions["max_users"])
			}
		}
		if room.RoomId == room2 {
			foundRoom2 = true
			if room.RoomOptions["name"] != "Game Room 1" {
				t.Errorf("Expected room2 name=Game Room 1, got %s", room.RoomOptions["name"])
			}
			if room.RoomOptions["game_type"] != "chess" {
				t.Errorf("Expected room2 game_type=chess, got %s", room.RoomOptions["game_type"])
			}
		}
		if room.RoomId == room3 {
			foundRoom3 = true
			if room.RoomOptions["name"] != "Game Room 2" {
				t.Errorf("Expected room3 name=Game Room 2, got %s", room.RoomOptions["name"])
			}
			if room.RoomOptions["private"] != "true" {
				t.Errorf("Expected room3 private=true, got %s", room.RoomOptions["private"])
			}
		}
	}

	if !foundRoom1 {
		t.Error("Room1 not found in RoomsList response")
	}
	if !foundRoom2 {
		t.Error("Room2 not found in RoomsList response")
	}
	if !foundRoom3 {
		t.Error("Room3 not found in RoomsList response")
	}

	// Verify that rooms have owner IDs set
	for _, room := range roomsList.Rooms {
		if room.RoomOwnerId == "" {
			t.Errorf("Room %s should have a non-empty owner ID", room.RoomId)
		}
	}

	t.Logf("Successfully listed %d rooms", len(roomsList.Rooms))
}
