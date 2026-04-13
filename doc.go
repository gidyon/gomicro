// Package gomicro provides lightweight helpers for building gRPC-first
// microservices with optional HTTP endpoints and grpc-gateway integration.
//
// The package centers around Service, which collects server options,
// interceptors, gateway options, HTTP middleware, and listeners before the
// process is started. Safe defaults are applied for common settings such as
// service name, ports, logger, and runtime mux path.
//
// Typical usage:
//
//  1. Create a service with NewService.
//  2. Add HTTP routes, gRPC interceptors, and dial/server options.
//  3. Register services on GRPCServer() and handlers on RuntimeMux().
//  4. Call Initialize or Start.
package gomicro
