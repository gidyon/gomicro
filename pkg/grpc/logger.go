package middleware

import (
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// codeToLevel maps gRPC status codes to zap log levels.
func codeToLevel(code codes.Code) zapcore.Level {
	return grpc_zap.DefaultCodeToLevel(code)
}

// AddLogging returns unary and stream interceptors for structured gRPC logging.
func AddLogging(
	logger *zap.Logger,
) ([]grpc.UnaryServerInterceptor, []grpc.StreamServerInterceptor) {
	if logger == nil {
		logger = zap.NewNop()
	}

	// Shared options for the logger, with a custom gRPC code to log level function.
	o := []grpc_zap.Option{
		grpc_zap.WithLevels(codeToLevel),
	}

	// Make sure that log statements internal to gRPC library are logged using the zapLogger as well.
	// grpc_zap.ReplaceGrpcLoggerV2(logger)

	// Add unary interceptors
	unaryInterceptors := []grpc.UnaryServerInterceptor{
		grpc_ctxtags.UnaryServerInterceptor(
			grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor),
		),
		grpc_zap.UnaryServerInterceptor(logger, o...),
	}

	// Add stream interceptors
	streamInterceptors := []grpc.StreamServerInterceptor{
		grpc_ctxtags.StreamServerInterceptor(
			grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor),
		),
		grpc_zap.StreamServerInterceptor(logger, o...),
	}

	return unaryInterceptors, streamInterceptors
}
