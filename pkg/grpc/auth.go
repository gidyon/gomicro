package middleware

import (
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"google.golang.org/grpc"
)

// AddAuth returns grpc.Server config option that turn on logging.
func AddAuth(
	authFunc grpc_auth.AuthFunc,
) ([]grpc.UnaryServerInterceptor, []grpc.StreamServerInterceptor) {
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
