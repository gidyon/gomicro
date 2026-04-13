package gomicro

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
)

func TestNewServiceAppliesDefaults(t *testing.T) {
	svc, err := NewService(nil)
	if err != nil {
		t.Fatalf("NewService(nil) error = %v", err)
	}

	if svc.options.ServiceName != defaultServiceName {
		t.Fatalf("ServiceName = %q, want %q", svc.options.ServiceName, defaultServiceName)
	}
	if svc.options.HttpPort != defaultHTTPPort {
		t.Fatalf("HttpPort = %d, want %d", svc.options.HttpPort, defaultHTTPPort)
	}
	if svc.options.GrpcPort != defaultGRPCPort {
		t.Fatalf("GrpcPort = %d, want %d", svc.options.GrpcPort, defaultGRPCPort)
	}
	if svc.options.Logger == nil {
		t.Fatal("Logger is nil, want default logger")
	}
	if svc.nowFunc == nil {
		t.Fatal("nowFunc is nil, want default clock")
	}
}

func TestNewServicePreservesProvidedLogger(t *testing.T) {
	customLogger := NewLogger("svc", zerolog.DebugLevel)

	svc, err := NewService(&Options{
		ServiceName: "svc",
		Logger:      customLogger,
		NowFunc: func() time.Time {
			return time.Unix(123, 0)
		},
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if svc.options.Logger != customLogger {
		t.Fatal("NewService() replaced the provided logger")
	}
	if got := svc.nowFunc(); !got.Equal(time.Unix(123, 0)) {
		t.Fatalf("nowFunc() = %v, want %v", got, time.Unix(123, 0))
	}
}

func TestZeroValueServiceAddersAreSafe(t *testing.T) {
	var svc Service

	svc.AddHTTPMiddlewares(func(next http.Handler) http.Handler { return next })
	svc.AddGRPCDialOptions(grpc.WithInsecure())
	svc.AddGRPCServerOptions(grpc.EmptyServerOption{})
	svc.AddGRPCUnaryServerInterceptors(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		return handler(ctx, req)
	})
	svc.AddGRPCStreamServerInterceptors(func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, ss)
	})
	svc.AddGRPCUnaryClientInterceptors(func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		return invoker(ctx, method, req, reply, cc, opts...)
	})
	svc.AddGRPCStreamClientInterceptors(func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		return streamer(ctx, desc, cc, method, opts...)
	})

	if svc.options == nil {
		t.Fatal("zero-value service did not initialize options")
	}
	if svc.RuntimeMux() == nil {
		t.Fatal("RuntimeMux() returned nil")
	}
	if svc.GRPCServer() == nil {
		t.Fatal("GRPCServer() returned nil")
	}
}
