package gomicro

import (
	"testing"

	"github.com/rs/zerolog"
)

func TestNewLoggerDefaultsServiceNameAndVerbosity(t *testing.T) {
	lg, ok := NewLogger("", zerolog.WarnLevel).(*logger)
	if !ok {
		t.Fatal("NewLogger() did not return *logger")
	}

	if lg.V(int(zerolog.InfoLevel)) {
		t.Fatal("V(info) = true, want false for warn-level logger")
	}
	if !lg.V(int(zerolog.ErrorLevel)) {
		t.Fatal("V(error) = false, want true for warn-level logger")
	}
}

func TestLoggerVerbosityNilReceiver(t *testing.T) {
	var lg *logger
	if lg.V(0) {
		t.Fatal("nil logger receiver should report V=false")
	}
}
