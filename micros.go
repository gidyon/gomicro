package gomicro

import (
	"context"
	"sync"
	"time"

	// "bitbucket.org/onfon/gomicro/pkg/config"

	"github.com/rs/zerolog"
	"google.golang.org/grpc/grpclog"

	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

// Service collects the state needed to bootstrap and run a gRPC service with
// optional HTTP endpoints and grpc-gateway support.
type Service struct {
	options                  *Options
	clientConn               *grpc.ClientConn
	gRPCServer               *grpc.Server
	serveMuxOptions          []runtime.ServeMuxOption
	serverOptions            []grpc.ServerOption
	unaryInterceptors        []grpc.UnaryServerInterceptor
	streamInterceptors       []grpc.StreamServerInterceptor
	dialOptions              []grpc.DialOption
	unaryClientInterceptors  []grpc.UnaryClientInterceptor
	streamClientInterceptors []grpc.StreamClientInterceptor
	httpMiddlewares          []func(http.Handler) http.Handler
	httpMux                  *http.ServeMux
	runtimeMux               *runtime.ServeMux
	shutdowns                []func() error
	initOnceFn               *sync.Once
	runOnceFn                *sync.Once
	nowFunc                  func() time.Time
}

// Options configures a Service.
type Options struct {
	// ServiceName identifies the service in logs and defaults.
	ServiceName string
	// HttpPort is the listener port for HTTP traffic and for combined HTTP/gRPC
	// traffic when TLS is enabled.
	HttpPort int
	// GrpcPort is the dedicated listener port for insecure gRPC traffic.
	GrpcPort int
	// Logger is used for service lifecycle logging. If nil, a default logger is created.
	Logger grpclog.LoggerV2
	// RuntimeMuxEndpoint is the path prefix used when mounting the grpc-gateway mux.
	RuntimeMuxEndpoint      string
	ServerReadTimeout       time.Duration
	ServerWriteTimeout      time.Duration
	ServerReadHeaderTimeout time.Duration
	// NowFunc overrides the clock used internally by the service.
	NowFunc func() time.Time
	// TLSEnabled enables secure serving on the HTTP port.
	TLSEnabled bool
	// TlSCertFile is the server certificate path used when TLS is enabled.
	TlSCertFile string
	// TlSKeyFile is the private key path used when TLS is enabled.
	TlSKeyFile string
	// TLSServerName is used by the local gateway client when dialing the secure gRPC endpoint.
	TLSServerName string
}

const (
	defaultServiceName     = "app"
	defaultHTTPPort        = 8080
	defaultGRPCPort        = 50051
	defaultRuntimeMuxPath  = "/"
	defaultShutdownTimeout = 10 * time.Second
)

// NewService creates a Service with safe defaults for omitted options.
func NewService(opt *Options) (*Service, error) {
	opt = normalizeOptions(opt)

	svc := &Service{
		options:                  opt,
		httpMiddlewares:          make([]func(http.Handler) http.Handler, 0),
		httpMux:                  http.NewServeMux(),
		runtimeMux:               runtime.NewServeMux(),
		serveMuxOptions:          make([]runtime.ServeMuxOption, 0),
		serverOptions:            make([]grpc.ServerOption, 0),
		unaryInterceptors:        make([]grpc.UnaryServerInterceptor, 0),
		streamInterceptors:       make([]grpc.StreamServerInterceptor, 0),
		dialOptions:              make([]grpc.DialOption, 0),
		unaryClientInterceptors:  make([]grpc.UnaryClientInterceptor, 0),
		streamClientInterceptors: make([]grpc.StreamClientInterceptor, 0),
		shutdowns:                make([]func() error, 0),
		initOnceFn:               &sync.Once{},
		runOnceFn:                &sync.Once{},
		nowFunc:                  opt.NowFunc,
	}

	return svc, nil
}

// AddEndpoint registers the handler for the given pattern.
// If a handler already exists for pattern, Handle panics.
func (service *Service) AddEndpoint(pattern string, handler http.Handler) {
	if service.httpMux == nil {
		service.httpMux = http.NewServeMux()
	}
	service.httpMux.Handle(pattern, handler)
}

// AddEndpointFunc registers the handler function for the given pattern.
func (service *Service) AddEndpointFunc(pattern string, handleFunc http.HandlerFunc) {
	if service.httpMux == nil {
		service.httpMux = http.NewServeMux()
	}
	service.httpMux.HandleFunc(pattern, handleFunc)
}

// AddHTTPMiddlewares appends HTTP middleware in the order they should wrap handlers.
func (service *Service) AddHTTPMiddlewares(middlewares ...func(http.Handler) http.Handler) {
	service.ensureDefaults()
	service.httpMiddlewares = append(service.httpMiddlewares, middlewares...)
}

// AddGRPCDialOptions appends dial options used by the service's internal gRPC client.
func (service *Service) AddGRPCDialOptions(dialOptions ...grpc.DialOption) {
	service.ensureDefaults()
	service.dialOptions = append(service.dialOptions, dialOptions...)
}

// AddGRPCServerOptions appends server options used when constructing the gRPC server.
func (service *Service) AddGRPCServerOptions(serverOptions ...grpc.ServerOption) {
	service.ensureDefaults()
	service.serverOptions = append(service.serverOptions, serverOptions...)
}

// AddGRPCStreamServerInterceptors appends stream server interceptors.
func (service *Service) AddGRPCStreamServerInterceptors(
	streamInterceptors ...grpc.StreamServerInterceptor,
) {
	service.ensureDefaults()
	service.streamInterceptors = append(
		service.streamInterceptors, streamInterceptors...,
	)
}

// AddGRPCUnaryServerInterceptors appends unary server interceptors.
func (service *Service) AddGRPCUnaryServerInterceptors(
	unaryInterceptors ...grpc.UnaryServerInterceptor,
) {
	service.ensureDefaults()
	service.unaryInterceptors = append(
		service.unaryInterceptors, unaryInterceptors...,
	)
}

// AddGRPCStreamClientInterceptors appends stream client interceptors used by the internal gateway client.
func (service *Service) AddGRPCStreamClientInterceptors(
	streamInterceptors ...grpc.StreamClientInterceptor,
) {
	service.ensureDefaults()
	service.streamClientInterceptors = append(
		service.streamClientInterceptors, streamInterceptors...,
	)
}

// AddGRPCUnaryClientInterceptors appends unary client interceptors used by the internal gateway client.
func (service *Service) AddGRPCUnaryClientInterceptors(
	unaryInterceptors ...grpc.UnaryClientInterceptor,
) {
	service.ensureDefaults()
	service.unaryClientInterceptors = append(
		service.unaryClientInterceptors, unaryInterceptors...,
	)
}

// AddRuntimeMuxOptions appends grpc-gateway ServeMux options applied during initialization.
func (service *Service) AddRuntimeMuxOptions(serveMuxOptions ...runtime.ServeMuxOption) {
	service.ensureDefaults()
	if service.serveMuxOptions == nil {
		service.serveMuxOptions = make([]runtime.ServeMuxOption, 0)
	}
	service.serveMuxOptions = append(service.serveMuxOptions, serveMuxOptions...)
}

// RuntimeMux returns the grpc-gateway mux used by the reverse proxy handler.
// Register generated gateway handlers on the returned mux.
func (service *Service) RuntimeMux() *runtime.ServeMux {
	service.ensureDefaults()
	return service.runtimeMux
}

// ClientConn returns the internal gRPC client connection used by the gateway.
func (service *Service) ClientConn() *grpc.ClientConn {
	service.ensureDefaults()
	return service.clientConn
}

// GRPCServer returns the gRPC server instance for service registration.
func (service *Service) GRPCServer() *grpc.Server {
	service.ensureDefaults()
	return service.gRPCServer
}

func (service *Service) ensureDefaults() {
	if service.options == nil {
		service.options = normalizeOptions(nil)
	} else {
		service.options = normalizeOptions(service.options)
	}
	if service.httpMiddlewares == nil {
		service.httpMiddlewares = make([]func(http.Handler) http.Handler, 0)
	}
	if service.httpMux == nil {
		service.httpMux = http.NewServeMux()
	}
	if service.runtimeMux == nil {
		service.runtimeMux = runtime.NewServeMux()
	}
	if service.serveMuxOptions == nil {
		service.serveMuxOptions = make([]runtime.ServeMuxOption, 0)
	}
	if service.serverOptions == nil {
		service.serverOptions = make([]grpc.ServerOption, 0)
	}
	if service.unaryInterceptors == nil {
		service.unaryInterceptors = make([]grpc.UnaryServerInterceptor, 0)
	}
	if service.streamInterceptors == nil {
		service.streamInterceptors = make([]grpc.StreamServerInterceptor, 0)
	}
	if service.dialOptions == nil {
		service.dialOptions = make([]grpc.DialOption, 0)
	}
	if service.unaryClientInterceptors == nil {
		service.unaryClientInterceptors = make([]grpc.UnaryClientInterceptor, 0)
	}
	if service.streamClientInterceptors == nil {
		service.streamClientInterceptors = make([]grpc.StreamClientInterceptor, 0)
	}
	if service.shutdowns == nil {
		service.shutdowns = make([]func() error, 0)
	}
	if service.initOnceFn == nil {
		service.initOnceFn = &sync.Once{}
	}
	if service.runOnceFn == nil {
		service.runOnceFn = &sync.Once{}
	}
	if service.nowFunc == nil {
		service.nowFunc = time.Now
	}
	if service.gRPCServer == nil {
		service.gRPCServer = grpc.NewServer()
	}
}

func normalizeOptions(opt *Options) *Options {
	if opt == nil {
		opt = &Options{}
	}
	if opt.ServiceName == "" {
		opt.ServiceName = defaultServiceName
	}
	if opt.HttpPort == 0 {
		opt.HttpPort = defaultHTTPPort
	}
	if opt.GrpcPort == 0 {
		opt.GrpcPort = defaultGRPCPort
	}
	if opt.RuntimeMuxEndpoint == "" {
		opt.RuntimeMuxEndpoint = defaultRuntimeMuxPath
	}
	if opt.Logger == nil {
		opt.Logger = NewLogger(opt.ServiceName, zerolog.InfoLevel)
	}
	if opt.NowFunc == nil {
		opt.NowFunc = time.Now
	}
	if opt.TLSEnabled && opt.TLSServerName == "" {
		opt.TLSServerName = "localhost"
	}
	return opt
}

func (service *Service) closeClientConn() error {
	if service == nil || service.clientConn == nil {
		return nil
	}
	if service.clientConn.GetState() == connectivity.Shutdown {
		return nil
	}
	return service.clientConn.Close()
}

func normalizeContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
