package gomicro

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/gidyon/gomicro/pkg/conn"

	"github.com/gidyon/gomicro/utils/tlsutil"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc/reflection"

	"google.golang.org/grpc"
)

// panic after encountering first non nil error
func handleErrs(errs ...error) {
	for _, err := range errs {
		if err != nil {
			panic(err)
		}
	}
}

// initializes service without starting it.
func (service *Service) init(ctx context.Context) {
	service.initOnceFn.Do(func() {
		handleErrs(
			service.initGRPC(ctx),
		)
	})
}

// Initialize initializes service without starting it.
func (service *Service) Initialize(ctx context.Context) {
	service.init(ctx)
}

// Start opens connection to databases and external services, afterwards starting grpc and http server to serve requests.
func (service *Service) Start(ctx context.Context, initFn func() error) {
	service.init(ctx)
	handleErrs(initFn(), service.run(ctx))
}

// apply applies a chain of middleware in order
func apply(handler http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	if len(middlewares) < 1 {
		return handler
	}
	wrapped := handler
	for i := len(middlewares) - 1; i >= 0; i-- {
		wrapped = middlewares[i](wrapped)
	}
	return wrapped
}

// starts the servers
func (service *Service) run(ctx context.Context) error {
	fn := func() error {
		defer func() {
			var err error
			for _, shutdown := range service.shutdowns {
				err = shutdown()
				if err != nil {
					service.options.Logger.Errorln(err)
				}
			}
		}()

		// Handles grpc gateway apis
		service.AddEndpoint(service.options.RuntimeMuxEndpoint, service.runtimeMux)

		// Apply any middlewares to the handler
		handler := apply(service.httpMux, service.httpMiddlewares...)

		var ghandler http.Handler

		// add grpc handler if TLS is enabled on service, will use same port
		if service.options.TLSEnabled {
			ghandler = grpcHandlerFunc(service.GRPCServer(), handler)
		} else {
			ghandler = handler
		}

		httpServer := &http.Server{
			Addr:         fmt.Sprintf(":%d", service.options.HttpPort),
			Handler:      ghandler,
			ReadTimeout:  service.options.ServerReadTimeout,
			WriteTimeout: service.options.ServerWriteTimeout,
		}

		// Graceful shutdown of server
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		go func() {
			for range c {
				service.options.Logger.Warning("shutting down service ...")
				service.gRPCServer.Stop()
				log.Fatalln(httpServer.Shutdown(ctx))

				<-ctx.Done()
			}
		}()

		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", service.options.HttpPort))
		if err != nil {
			return fmt.Errorf("failed to create TCP listener for http server: %v", err)
		}
		defer lis.Close()

		logMsgFn := func() {
			if !service.options.TLSEnabled {
				service.options.Logger.Infof(
					"<GRPC> running on port %d (insecure), <REST> server running on port %d (insecure)",
					service.options.GrpcPort, service.options.HttpPort,
				)
			} else {
				service.options.Logger.Infof("<gRPC> and <REST> server running on same port %d (secure)", service.options.HttpPort)
			}
		}

		logMsgFn()

		if !service.options.TLSEnabled {
			glis, err := net.Listen("tcp", fmt.Sprintf(":%d", service.options.GrpcPort))
			if err != nil {
				return fmt.Errorf("failed to create TCP listener for gRPC server: %v", err)
			}
			defer glis.Close()

			// Note: The call to serve grpc must be inside a goroutine; don't do [go service.gRPCServer.Serve(glis)]
			go func() {
				err := service.gRPCServer.Serve(glis)
				if err != nil {
					service.options.Logger.Errorln(err)
				}
			}()

			// Serve http insecurely
			return httpServer.Serve(lis)
		}

		// Get PK for server
		cert, certPool, err := tlsutil.GetCert(service.options.TlSCertFile, service.options.TlSKeyFile)
		if err != nil {
			return err
		}

		// Create tls object
		tlsConfig := &tls.Config{
			NextProtos:         []string{"h2", "http/1.1", "http/1.2"},
			MinVersion:         tls.VersionTLS10,
			MaxVersion:         tls.VersionTLS13,
			ClientAuth:         tls.VerifyClientCertIfGiven,
			ClientCAs:          certPool,
			Certificates:       []tls.Certificate{*cert},
			InsecureSkipVerify: true,
		}

		// Serve tls
		return httpServer.Serve(tls.NewListener(lis, tlsConfig))
	}

	var err error
	service.runOnceFn.Do(func() {
		err = fn()
	})

	return err
}

// grpcHandlerFunc returns an http.Handler that delegates to grpcServer on incoming gRPC
// connections or otherHandler otherwise.
func grpcHandlerFunc(grpcServer *grpc.Server, otherHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			otherHandler.ServeHTTP(w, r)
		}
	})
}

// initGRPC initialize gRPC server and client with registered client and server interceptors and options.
// The method must be called before registering anything on the gRPC server or passing options to the gRPC client.
// When this method has been called, subsequent calls to update interceptors becomes stale.
func (service *Service) initGRPC(ctx context.Context) error {
	// ============================= Update runtime mux endpoint =============================
	if service.options.RuntimeMuxEndpoint == "" {
		service.options.RuntimeMuxEndpoint = "/"
	}

	// Apply servemux options to runtime muxer
	service.runtimeMux = runtime.NewServeMux(service.serveMuxOptions...)

	// ============================= Initialize grpc proxy client =============================
	var (
		gPort int
		err   error
	)

	if service.options.TLSEnabled {
		creds, err := credentials.NewClientTLSFromFile(service.options.TlSCertFile, service.options.TLSServerName)
		if err != nil {
			return fmt.Errorf("failed to create tls config for %s service: %v", service.options.TLSServerName, err)
		}
		service.dialOptions = append(service.dialOptions, grpc.WithTransportCredentials(creds))
		gPort = service.options.HttpPort
	} else {
		service.dialOptions = append(service.dialOptions, grpc.WithTransportCredentials(insecure.NewCredentials()))
		gPort = service.options.GrpcPort
	}

	// Enable wait for ready RPCs
	waitForReadyUnaryInterceptor := func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		return invoker(ctx, method, req, reply, cc, append(opts, grpc.WaitForReady(true))...)
	}

	// Add client unary interceptos
	unaryClientInterceptors := []grpc.UnaryClientInterceptor{waitForReadyUnaryInterceptor}
	unaryClientInterceptors = append(unaryClientInterceptors, service.unaryClientInterceptors...)

	// Add client streaming interceptos
	streamClientInterceptors := make([]grpc.StreamClientInterceptor, 0)
	streamClientInterceptors = append(streamClientInterceptors, service.streamClientInterceptors...)

	// Add inteceptors as dial option
	service.dialOptions = append(service.dialOptions, []grpc.DialOption{
		grpc.WithUnaryInterceptor(
			grpc_middleware.ChainUnaryClient(unaryClientInterceptors...),
		),
		grpc.WithStreamInterceptor(
			grpc_middleware.ChainStreamClient(streamClientInterceptors...),
		),
	}...)

	// client connection to the reverse gateway
	service.clientConn, err = conn.DialGrpcService(context.Background(), &conn.GrpcDialOptions{
		ServiceName: "self",
		Address:     fmt.Sprintf("localhost:%d", gPort),
		DialOptions: service.dialOptions,
		K8Service:   false,
	})
	if err != nil {
		return fmt.Errorf("client failed to dial to gRPC server: %v", err)
	}

	// ============================= Initialize grpc server =============================
	// Add transport credentials if secure option is passed
	if service.options.TLSEnabled {
		creds, err := credentials.NewServerTLSFromFile(service.options.TlSCertFile, service.options.TlSKeyFile)
		if err != nil {
			return fmt.Errorf("failed to create grpc server tls credentials: %v", err)
		}
		service.serverOptions = append(
			service.serverOptions, grpc.Creds(creds),
		)
	}

	// Append interceptors as server options
	service.serverOptions = append(
		service.serverOptions, grpc_middleware.WithUnaryServerChain(service.unaryInterceptors...))
	service.serverOptions = append(
		service.serverOptions, grpc_middleware.WithStreamServerChain(service.streamInterceptors...))

	service.gRPCServer = grpc.NewServer(service.serverOptions...)

	// register reflection on the gRPC server
	reflection.Register(service.gRPCServer)

	return nil
}
