package middleware

import (
	"context"

	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func customFunc(ctx context.Context, p interface{}) error {
	return status.Errorf(codes.Internal, "recovered from panic: %v", p)
}

// AddRecovery returns interceptors that convert panics in gRPC handlers into internal errors.
func AddRecovery() ([]grpc.UnaryServerInterceptor, []grpc.StreamServerInterceptor) {
	// Shared option for the logger, with a custom gRPC code to log level function.
	opt := grpc_recovery.WithRecoveryHandlerContext(customFunc)

	// Recovery handlers should typically be last in the chain so that other middleware
	// (e.g. logging) can operate on the recovered state instead of being directly affected by any panic
	return []grpc.UnaryServerInterceptor{
			grpc_recovery.UnaryServerInterceptor(opt),
		}, []grpc.StreamServerInterceptor{
			grpc_recovery.StreamServerInterceptor(opt),
		}
}
