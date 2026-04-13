package conn

import (
	"context"
	"strings"
	"testing"

	"google.golang.org/grpc"
)

func TestDialGrpcServiceValidation(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		opt     *GrpcDialOptions
		wantErr string
	}{
		{name: "nil context", opt: &GrpcDialOptions{Address: "127.0.0.1:8080"}, wantErr: "nil context"},
		{name: "nil options", ctx: context.Background(), wantErr: "nil grpc dial options"},
		{name: "missing address", ctx: context.Background(), opt: &GrpcDialOptions{}, wantErr: "grpc service address is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, err := DialGrpcService(tt.ctx, tt.opt)
			if conn != nil {
				t.Fatal("DialGrpcService() returned unexpected connection")
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("DialGrpcService() error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestDialGrpcServiceDoesNotMutateAddress(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	opt := &GrpcDialOptions{
		Address:     "service:50051",
		DialOptions: []grpc.DialOption{grpc.WithBlock()},
	}

	_, err := DialGrpcService(ctx, opt)
	if err == nil {
		t.Fatal("DialGrpcService() expected error from canceled context")
	}
	if opt.Address != "service:50051" {
		t.Fatalf("DialGrpcService() mutated address to %q", opt.Address)
	}
}
