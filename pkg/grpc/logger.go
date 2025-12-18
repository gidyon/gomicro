package middleware

import (
	"context"

	grpc_logging "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// codeToLevel matches old zap behavior: OK → debug instead of info
func codeToLevel(code codes.Code) grpc_logging.Level {
	return grpc_logging.DefaultServerCodeToLevel(code)
}

// AddLogging returns grpc.Server config option that turn on logging.
func AddLogging(
	logger *zap.Logger,
) ([]grpc.UnaryServerInterceptor, []grpc.StreamServerInterceptor) {

	// Adapt zap.Logger to v2 interceptor Logger
	interceptorLogger := grpc_logging.LoggerFunc(func(ctx context.Context, lvl grpc_logging.Level, msg string, fields ...any) {

		f := make([]zap.Field, 0, len(fields)/2)

		for i := 0; i < len(fields); i += 2 {
			k := fields[i]
			v := fields[i+1]
			key := k.(string)

			switch x := v.(type) {
			case string:
				f = append(f, zap.String(key, x))
			case int:
				f = append(f, zap.Int(key, x))
			case bool:
				f = append(f, zap.Bool(key, x))
			default:
				f = append(f, zap.Any(key, x))
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
		default:
			l.Error("unknown log level", zap.String("msg", msg))
		}
	})

	opts := []grpc_logging.Option{
		grpc_logging.WithLogOnEvents(
			grpc_logging.StartCall,
			grpc_logging.FinishCall,
		),
		grpc_logging.WithLevels(codeToLevel),
	}

	// Unary interceptors
	unaryInterceptors := []grpc.UnaryServerInterceptor{
		grpc_logging.UnaryServerInterceptor(interceptorLogger, opts...),
	}

	// Stream interceptors
	streamInterceptors := []grpc.StreamServerInterceptor{
		grpc_logging.StreamServerInterceptor(interceptorLogger, opts...),
	}

	return unaryInterceptors, streamInterceptors
}
