// Package conn contains connection helpers for the most common outbound
// dependencies used by gomicro services.
//
// The package currently includes:
//
// - MySQL connection helpers for database/sql and GORM.
// - gRPC dial helpers with wait-for-ready behavior and resolver prefix handling.
//
// The exported constructors validate their inputs so configuration problems fail
// fast instead of surfacing later as confusing connection errors.
package conn
