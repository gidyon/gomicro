package middleware

import (
	"context"
	"fmt"

	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"google.golang.org/grpc"
)

// When panic happens, convert panic → gRPC error.
func customFunc(ctx context.Context, p interface{}) error {
	return fmt.Errorf("recovering from panic: %v", p)
}

// AddRecovery recovers from gRPC handler panics.
func AddRecovery() ([]grpc.UnaryServerInterceptor, []grpc.StreamServerInterceptor) {

	opts := []grpc_recovery.Option{
		grpc_recovery.WithRecoveryHandlerContext(customFunc),
	}

	unary := []grpc.UnaryServerInterceptor{
		grpc_recovery.UnaryServerInterceptor(opts...),
	}

	stream := []grpc.StreamServerInterceptor{
		grpc_recovery.StreamServerInterceptor(opts...),
	}

	return unary, stream
}
