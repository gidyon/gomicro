// Package middleware provides reusable gRPC server interceptors for common
// production concerns such as authentication, structured logging, payload
// logging, and panic recovery.
//
// The helper functions return interceptor slices so callers can compose them
// into their own preferred chain order.
package middleware
