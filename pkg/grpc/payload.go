package middleware

import (
	"context"

	grpc_logging "github.com/grpc-ecosystem/go-grpc-middleware/logging"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// AddPayloadLogging returns interceptors that log request and response payloads
// in addition to standard gRPC call metadata.
func AddPayloadLogging(
	logger *zap.Logger,
) ([]grpc.UnaryServerInterceptor, []grpc.StreamServerInterceptor) {
	if logger == nil {
		logger = zap.NewNop()
	}
	decider := func(ctx context.Context, fullMethodName string, servingObject interface{}) bool {
		return grpc_logging.DefaultDeciderMethod(fullMethodName, nil)
	}

	// Add unary interceptors
	unaryInterceptors := []grpc.UnaryServerInterceptor{
		grpc_ctxtags.UnaryServerInterceptor(
			grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor),
		),
		grpc_zap.UnaryServerInterceptor(logger, grpc_zap.WithLevels(codeToLevel)),
		grpc_zap.PayloadUnaryServerInterceptor(logger, decider),
	}

	// Add stream interceptors
	streamInterceptors := []grpc.StreamServerInterceptor{
		grpc_ctxtags.StreamServerInterceptor(
			grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor),
		),
		grpc_zap.StreamServerInterceptor(logger, grpc_zap.WithLevels(codeToLevel)),
		grpc_zap.PayloadStreamServerInterceptor(logger, decider),
	}

	return unaryInterceptors, streamInterceptors
}
