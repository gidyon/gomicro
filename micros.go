package gomicro

import (
	"sync"
	"time"

	// "bitbucket.org/onfon/gomicro/pkg/config"

	http_middleware "github.com/gidyon/gomicro/pkg/http"

	"github.com/rs/zerolog"
	"google.golang.org/grpc/grpclog"

	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

// Service contains API clients, connections and options for bootstrapping a micro-service.
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
	httpMiddlewares          []http_middleware.Middleware
	httpMux                  *http.ServeMux
	runtimeMux               *runtime.ServeMux
	shutdowns                []func() error
	initOnceFn               *sync.Once
	runOnceFn                *sync.Once
	nowFunc                  func() time.Time
}

type Options struct {
	ServiceName        string
	HttpPort           int
	GrpcPort           int
	Logger             grpclog.LoggerV2
	RuntimeMuxEndpoint string
	ServerReadTimeout  time.Duration
	ServerWriteTimeout time.Duration
	NowFunc            func() time.Time
	TLSEnabled         bool
	TlSCertFile        string
	TlSKeyFile         string
	TLSServerName      string
}

// NewService create a micro-service utility store by parsing data from config. Pass nil logger to use default logger
func NewService(opt *Options) (*Service, error) {
	if opt.Logger != nil {
		opt.Logger = NewLogger("app", zerolog.TraceLevel)
	}

	svc := &Service{
		options:                  opt,
		httpMiddlewares:          make([]http_middleware.Middleware, 0),
		httpMux:                  http.NewServeMux(),
		runtimeMux:               runtime.NewServeMux(),
		clientConn:               &grpc.ClientConn{},
		gRPCServer:               &grpc.Server{},
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

// AddHTTPMiddlewares adds http middlewares to the service
func (service *Service) AddHTTPMiddlewares(middlewares ...http_middleware.Middleware) {
	service.httpMiddlewares = append(service.httpMiddlewares, middlewares...)
}

// AddGRPCDialOptions adds dial options to the service gRPC reverse proxy client
func (service *Service) AddGRPCDialOptions(dialOptions ...grpc.DialOption) {
	service.dialOptions = append(service.dialOptions, dialOptions...)
}

// AddGRPCServerOptions adds server options to the service gRPC server
func (service *Service) AddGRPCServerOptions(serverOptions ...grpc.ServerOption) {
	service.serverOptions = append(service.serverOptions, serverOptions...)
}

// AddGRPCStreamServerInterceptors adds stream interceptors to the service gRPC server
func (service *Service) AddGRPCStreamServerInterceptors(
	streamInterceptors ...grpc.StreamServerInterceptor,
) {
	service.streamInterceptors = append(
		service.streamInterceptors, streamInterceptors...,
	)
}

// AddGRPCUnaryServerInterceptors adds unary interceptors to the service gRPC server
func (service *Service) AddGRPCUnaryServerInterceptors(
	unaryInterceptors ...grpc.UnaryServerInterceptor,
) {
	service.unaryInterceptors = append(
		service.unaryInterceptors, unaryInterceptors...,
	)
}

// AddGRPCStreamClientInterceptors adds stream interceptors to the service gRPC reverse proxy client
func (service *Service) AddGRPCStreamClientInterceptors(
	streamInterceptors ...grpc.StreamClientInterceptor,
) {
	service.streamClientInterceptors = append(
		service.streamClientInterceptors, streamInterceptors...,
	)
}

// AddGRPCUnaryClientInterceptors adds unary interceptors to the service gRPC reverse proxy client
func (service *Service) AddGRPCUnaryClientInterceptors(
	unaryInterceptors ...grpc.UnaryClientInterceptor,
) {
	service.unaryClientInterceptors = append(
		service.unaryClientInterceptors, unaryInterceptors...,
	)
}

// AddRuntimeMuxOptions adds ServeMuxOption options to service gRPC reverse proxy client
// The options will be applied to the service runtime mux at startup
func (service *Service) AddRuntimeMuxOptions(serveMuxOptions ...runtime.ServeMuxOption) {
	if service.serveMuxOptions == nil {
		service.serveMuxOptions = make([]runtime.ServeMuxOption, 0)
	}
	service.serveMuxOptions = append(service.serveMuxOptions, serveMuxOptions...)
}

// RuntimeMux returns the HTTP request multiplexer for the service reverse proxy server
// gRPC services and methods are registered on this multiplxer.
// DO NOT register your anything on the returned muxer
// Use AddRuntimeMuxOptions method to register custom options
func (service *Service) RuntimeMux() *runtime.ServeMux {
	return service.runtimeMux
}

// ClientConn returns the underlying client connection to gRPC server used by reverse proxy
func (service *Service) ClientConn() *grpc.ClientConn {
	return service.clientConn
}

// GRPCServer returns the grpc server for the service
func (service *Service) GRPCServer() *grpc.Server {
	return service.gRPCServer
}
