# gomicro

`gomicro` provides small building blocks for Go microservices that expose gRPC APIs, optional HTTP gateways, structured logging, JWT-based authentication helpers, and connection utilities.

The repository is intentionally lightweight: you assemble the pieces you need instead of adopting a large framework.

## Packages

- `github.com/gidyon/gomicro`: service bootstrap, gRPC/HTTP startup, and logging helpers.
- `github.com/gidyon/gomicro/pkg/conn`: SQL and gRPC client connection helpers.
- `github.com/gidyon/gomicro/pkg/grpc`: reusable gRPC server middleware for auth, logging, payload logging, and panic recovery.
- `github.com/gidyon/gomicro/pkg/grpc/auth`: JWT authentication and authorization helpers for gRPC services.

## Quick Start

Create a service and register plain HTTP endpoints:

```go
svc, err := gomicro.NewService(&gomicro.Options{
    ServiceName: "users",
    HttpPort:    8080,
    GrpcPort:    9090,
})
if err != nil {
    return err
}

svc.AddEndpointFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    _, _ = w.Write([]byte("ok"))
})
```

Add gRPC middleware before initialization:

```go
loggerInterceptors, streamLoggerInterceptors := middleware.AddLogging(zap.NewNop())
svc.AddGRPCUnaryServerInterceptors(loggerInterceptors...)
svc.AddGRPCStreamServerInterceptors(streamLoggerInterceptors...)

recoveryUnary, recoveryStream := middleware.AddRecovery()
svc.AddGRPCUnaryServerInterceptors(recoveryUnary...)
svc.AddGRPCStreamServerInterceptors(recoveryStream...)
```

Initialize JWT auth for a gRPC service:

```go
authAPI := grpcauth.NewAPI([]byte("secret"), "users", "users-api")

token, err := authAPI.GenToken(context.Background(), &grpcauth.Payload{
    ID:        "user-123",
    ProjectID: "project-1",
    Group:     grpcauth.DefaultUserGroup(),
}, time.Now().Add(time.Hour))
if err != nil {
    return err
}

md, _ := authAPI.GetMetadataFromJwt(token)
_ = md
```

Dial a gRPC service:

```go
conn, err := conn.DialGrpcService(context.Background(), &conn.GrpcDialOptions{
    Address: "localhost:9090",
    DialOptions: []grpc.DialOption{
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    },
})
if err != nil {
    return err
}
defer conn.Close()
```

Open a MySQL connection with pool settings:

```go
db, err := conn.OpenSql(&conn.DbOptions{
    Name:    "users",
    Address: "127.0.0.1:3306",
    User:    "root",
    Schema:  "users",
    ConnPool: &conn.DbPoolSettings{
        MaxIdleConns: 5,
        MaxOpenConns: 20,
    },
})
if err != nil {
    return err
}
defer db.Close()
```

## Lifecycle Notes

- Add server/client interceptors and runtime mux options before calling `Initialize` or `Start`.
- `NewService` applies safe defaults when options are omitted.
- When TLS is enabled, provide both `TlSCertFile` and `TlSKeyFile`.
- `Start` blocks until the service stops; use `Initialize` if you need to prepare the service before starting it elsewhere.

## Testing

Run the full suite with:

```bash
GOCACHE=/tmp/go-build go test ./...
```
