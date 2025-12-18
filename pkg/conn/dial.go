package conn

import (
	"context"
	"strings"

	"google.golang.org/grpc"
)

// GrpcDialOptions contains options for dialing a grpc service
type GrpcDialOptions struct {
	ServiceName string
	Address     string
	DialOptions []grpc.DialOption
	K8Service   bool
}

func DialGrpcService(ctx context.Context, opt *GrpcDialOptions) (*grpc.ClientConn, error) {

	// base client opts
	dopts := []grpc.DialOption{
		grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [ { "round_robin": {} } ] }`),

		grpc.WithDisableServiceConfig(),

		grpc.WithChainUnaryInterceptor(
			waitForReadyUnaryInterceptor,
		),
	}

	// add user dial options
	dopts = append(dopts, opt.DialOptions...)

	// rewrite address format
	if opt.K8Service {
		opt.Address = strings.TrimSuffix(opt.Address, "dns:///")
		opt.Address = "dns:///" + opt.Address
	} else {
		opt.Address = strings.TrimSuffix(opt.Address, "passthrough:///")
		opt.Address = "passthrough:///" + opt.Address
	}

	return grpc.NewClient(opt.Address, dopts...)
}

func waitForReadyUnaryInterceptor(
	ctx context.Context,
	method string,
	req, reply interface{},
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	return invoker(ctx, method, req, reply, cc,
		append(opts, grpc.WaitForReady(true))...,
	)
}
