package middleware

import (
	"context"

	grpc_logging "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func AddPayloadLogging(
	logger *zap.Logger,
) ([]grpc.UnaryServerInterceptor, []grpc.StreamServerInterceptor) {

	// zap adapter → v2 logging.Logger
	interceptorLogger := grpc_logging.LoggerFunc(func(ctx context.Context, lvl grpc_logging.Level, msg string, fields ...any) {

		f := make([]zap.Field, 0, len(fields)/2)

		for i := 0; i < len(fields); i += 2 {
			key := fields[i].(string)
			val := fields[i+1]

			switch v := val.(type) {
			case string:
				f = append(f, zap.String(key, v))
			case bool:
				f = append(f, zap.Bool(key, v))
			case int:
				f = append(f, zap.Int(key, v))
			default:
				f = append(f, zap.Any(key, v))
			}
		}

		l := logger.WithOptions(zap.AddCallerSkip(1)).With(f...)

		switch lvl {
		case grpc_logging.LevelDebug:
			l.Debug(msg)
		case grpc_logging.LevelInfo:
			l.Info(msg)
		case grpc_logging.LevelWarn:
			l.Warn(msg)
		case grpc_logging.LevelError:
			l.Error(msg)
		}
	})

	opts := []grpc_logging.Option{
		grpc_logging.WithLevels(codeToLevel),

		// ⭐ PAYLOAD logging here:
		grpc_logging.WithLogOnEvents(
			grpc_logging.PayloadReceived,
			grpc_logging.PayloadSent,
		),
	}

	unary := []grpc.UnaryServerInterceptor{
		grpc_logging.UnaryServerInterceptor(interceptorLogger, opts...),
	}

	stream := []grpc.StreamServerInterceptor{
		grpc_logging.StreamServerInterceptor(interceptorLogger, opts...),
	}

	return unary, stream
}
