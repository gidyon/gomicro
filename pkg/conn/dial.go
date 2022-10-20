package conn

import (
	"context"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"

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

// DialGrpcService dials to a grpc service
func DialGrpcService(ctx context.Context, opt *GrpcDialOptions) (*grpc.ClientConn, error) {
	var (
		dopts = []grpc.DialOption{
			grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [ { "round_robin": {} } ] }`),
			// Load balancer scheme
			grpc.WithDisableServiceConfig(),
			// Other interceptors
			grpc.WithUnaryInterceptor(
				grpc_middleware.ChainUnaryClient(
					waitForReadyInterceptor,
				),
			),
		}
	)

	dopts = append(dopts, opt.DialOptions...)

	// Address for dialing the kubernetes service
	if opt.K8Service {
		opt.Address = strings.TrimSuffix(opt.Address, "dns:///")
		opt.Address = "dns:///" + opt.Address
	} else {
		opt.Address = strings.TrimSuffix(opt.Address, "passthrough:///")
		opt.Address = "passthrough:///" + opt.Address
	}

	return grpc.DialContext(ctx, opt.Address, dopts...)
}

func waitForReadyInterceptor(
	ctx context.Context,
	method string,
	req, reply interface{},
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	return invoker(ctx, method, req, reply, cc, append(opts, grpc.WaitForReady(true))...)
}
