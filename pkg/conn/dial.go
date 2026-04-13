package conn

import (
	"context"
	"errors"
	"strings"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"

	"google.golang.org/grpc"
)

// GrpcDialOptions controls how DialGrpcService connects to a gRPC service.
type GrpcDialOptions struct {
	ServiceName string
	// Address is the host:port pair or resolver target to dial.
	Address string
	// DialOptions are appended after gomicro's default wait-for-ready and balancing options.
	DialOptions []grpc.DialOption
	// K8Service switches the resolver prefix to dns:/// for Kubernetes service discovery.
	K8Service bool
}

// DialGrpcService dials a gRPC service using gomicro defaults plus any caller-provided options.
func DialGrpcService(ctx context.Context, opt *GrpcDialOptions) (*grpc.ClientConn, error) {
	if ctx == nil {
		return nil, errors.New("nil context not allowed")
	}
	if opt == nil {
		return nil, errors.New("nil grpc dial options not allowed")
	}
	if strings.TrimSpace(opt.Address) == "" {
		return nil, errors.New("grpc service address is required")
	}

	var (
		dopts = []grpc.DialOption{
			grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [ { "round_robin": {} } ] }`),
			// Other interceptors
			grpc.WithUnaryInterceptor(
				grpc_middleware.ChainUnaryClient(
					waitForReadyInterceptor,
				),
			),
		}
	)

	dopts = append(dopts, opt.DialOptions...)

	address := opt.Address

	// Address for dialing the kubernetes service
	if opt.K8Service {
		address = strings.TrimPrefix(address, "dns:///")
		address = "dns:///" + address
	} else {
		address = strings.TrimPrefix(address, "passthrough:///")
		address = "passthrough:///" + address
	}

	return grpc.DialContext(ctx, address, dopts...)
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
