package middleware

import (
	"context"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestAddAuthNilFunc(t *testing.T) {
	unary, stream := AddAuth(nil)
	if unary != nil || stream != nil {
		t.Fatalf("AddAuth(nil) = (%v, %v), want nil interceptors", unary, stream)
	}
}

func TestAddAuthInterceptorRejectsMissingMetadata(t *testing.T) {
	unary, _ := AddAuth(func(ctx context.Context) (context.Context, error) {
		return nil, status.Error(codes.PermissionDenied, "denied")
	})
	if len(unary) != 1 {
		t.Fatalf("expected 1 unary interceptor, got %d", len(unary))
	}

	_, err := unary[0](context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/svc/method"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.PermissionDenied)
	}
}

func TestAddLoggingNilLogger(t *testing.T) {
	unary, stream := AddLogging(nil)
	if len(unary) != 2 || len(stream) != 2 {
		t.Fatalf("AddLogging(nil) returned (%d, %d) interceptors, want (2, 2)", len(unary), len(stream))
	}
}

func TestAddPayloadLoggingIncludesPayloadInterceptors(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)

	unary, stream := AddPayloadLogging(logger)
	if len(unary) != 3 || len(stream) != 3 {
		t.Fatalf("AddPayloadLogging() returned (%d, %d) interceptors, want (3, 3)", len(unary), len(stream))
	}

	resp, err := unary[2](context.Background(), &emptypb.Empty{}, &grpc.UnaryServerInfo{
		FullMethod: "/svc/method",
		Server:     struct{}{},
	}, func(ctx context.Context, req interface{}) (interface{}, error) {
		return &emptypb.Empty{}, nil
	})
	if err != nil {
		t.Fatalf("payload unary interceptor error = %v", err)
	}
	if _, ok := resp.(*emptypb.Empty); !ok {
		t.Fatalf("payload unary interceptor returned %T", resp)
	}

	if logs.Len() != 2 {
		t.Fatalf("observed %d log entries, want 2", logs.Len())
	}
}

func TestAddRecoveryRecoversPanic(t *testing.T) {
	unary, stream := AddRecovery()
	if len(unary) != 1 || len(stream) != 1 {
		t.Fatalf("AddRecovery() returned (%d, %d) interceptors, want (1, 1)", len(unary), len(stream))
	}

	_, err := unary[0](context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/svc/method"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		panic("boom")
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
	if !strings.Contains(err.Error(), "recovered from panic") {
		t.Fatalf("recovery error = %v, want panic message", err)
	}
}

func TestCodeToLevelUsesDefaultMapping(t *testing.T) {
	if got := codeToLevel(codes.OK); got != zapcore.InfoLevel {
		t.Fatalf("codeToLevel(codes.OK) = %v, want %v", got, zapcore.InfoLevel)
	}
}
