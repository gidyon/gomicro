package zaplogger

import (
	"testing"

	"go.uber.org/zap"
)

func TestZapGrpcLoggerV2HandlesNilLogger(t *testing.T) {
	logger := ZapGrpcLoggerV2(nil)
	if logger == nil {
		t.Fatal("ZapGrpcLoggerV2(nil) returned nil")
	}
	if !logger.V(0) {
		t.Fatal("expected default verbosity to allow level 0")
	}
}

func TestZapGrpcLoggerV2UsesProvidedLogger(t *testing.T) {
	logger := ZapGrpcLoggerV2(zap.NewNop())
	if logger == nil {
		t.Fatal("ZapGrpcLoggerV2() returned nil")
	}
}
