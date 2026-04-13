package middleware

import (
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"google.golang.org/grpc"
)

// AddAuth returns unary and stream interceptors that enforce gRPC auth middleware.
func AddAuth(
	authFunc grpc_auth.AuthFunc,
) ([]grpc.UnaryServerInterceptor, []grpc.StreamServerInterceptor) {
	if authFunc == nil {
		return nil, nil
	}

	// Add unary interceptors
	unaryInterceptors := []grpc.UnaryServerInterceptor{
		grpc_auth.UnaryServerInterceptor(authFunc),
	}

	// Add stream interceptors
	streamInterceptors := []grpc.StreamServerInterceptor{
		grpc_auth.StreamServerInterceptor(authFunc),
	}

	return unaryInterceptors, streamInterceptors
}
