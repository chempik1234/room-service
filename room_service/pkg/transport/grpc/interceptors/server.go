package interceptors

import (
	"google.golang.org/grpc"
)

// MiddlewaresForService returns unary and stream interceptors based on config
// Keeps main.go clean by centralizing interceptor logic
func MiddlewaresForService(useAuth bool) ([]grpc.UnaryServerInterceptor, []grpc.StreamServerInterceptor) {
	var unaryInterceptors []grpc.UnaryServerInterceptor
	var streamInterceptors []grpc.StreamServerInterceptor

	// Always add logging first (executes first on request)
	unaryInterceptors = append(unaryInterceptors, AddLogMiddleware)
	streamInterceptors = append(streamInterceptors, AddLogMiddlewareStream)

	// Add auth if enabled (executes after logging)
	if useAuth {
		unaryInterceptors = append(unaryInterceptors, APIKeyAuth)
		streamInterceptors = append(streamInterceptors, APIKeyAuthStream)
	}

	return unaryInterceptors, streamInterceptors
}
