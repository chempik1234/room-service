## Note: AI Generated

# Authentication

## Current Implementation

Room Service uses simple API key authentication for all gRPC methods.

**Algorithm**:
1. Client sends API key in gRPC metadata header: `x-api-key: apikey`
2. Server validates key matches hardcoded value: `"apikey"`
3. If valid → request proceeds
4. If invalid/missing → returns `Unauthenticated` status

## Configuration

Enable/disable authentication via environment variable:

```bash
ROOM_SERVICE_USE_AUTH=true  # Enable auth (default)
ROOM_SERVICE_USE_AUTH=false # Disable auth (for development)
```

## Usage Example (Go)

```go
import (
    "context"
    "google.golang.org/grpc"
    "google.golang.org/grpc/metadata"
)

conn, _ := grpc.Dial("localhost:50050", grpc.WithInsecure())
client := NewRoomServiceClient(conn)

// Add API key to metadata
md := metadata.Pairs("x-api-key", "apikey")
ctx := metadata.NewOutgoingContext(context.Background(), md)

// Make authenticated request
response, err := client.SingleCommand(ctx, &Command{...})
```

## Usage Example (Python)

```python
import grpc

metadata = [('x-api-key', 'apikey')]
channel = grpc.insecure_channel('localhost:50050')
stub = room_service_pb2_grpc.RoomServiceStub(channel)

response = stub.SingleCommand(
    request=room_service_pb2.Command(...),
    metadata=metadata
)
```

## Testing with grpcurl

```bash
# Without auth (will fail if auth is enabled)
grpcurl -plaintext localhost:50050 api.RoomService/SingleCommand

# With auth
grpcurl -plaintext \
    -H "x-api-key: apikey" \
    -d '{"command_id": "test"}' \
    localhost:50050 api.RoomService/SingleCommand
```

## TODO: Proper API Key Storage

Current implementation uses hardcoded API key `"apikey"` for simplicity.

**Planned improvements**:
- [ ] Store API keys in Redis
- [ ] API key generation utility (random, prefixed)
- [ ] API key management CLI (create, delete, list)
- [ ] Per-key metadata (creation date, description)
- [ ] API key expiration/TTL
- [ ] Admin gRPC service for key management
- [ ] API key rotation support

**Storage design**:
```
Redis key pattern: api_key:{key}
Redis value: "active" (or JSON with metadata)
```

## Security Notes

- API key is sent in plaintext (consider TLS for production)
- Same key for all clients (temporary, will be replaced)
- No rate limiting (add later)
- No key rotation (add later)
- Auth applies to all methods when enabled (no per-method exceptions yet)
- Authentication works for both unary and streaming RPCs
