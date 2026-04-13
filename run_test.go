package gomicro

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestApplyMiddlewareOrder(t *testing.T) {
	order := make([]string, 0, 4)
	handler := apply(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
	}), func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw1-before")
			next.ServeHTTP(w, r)
			order = append(order, "mw1-after")
		})
	}, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw2-before")
			next.ServeHTTP(w, r)
			order = append(order, "mw2-after")
		})
	})

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	got := strings.Join(order, ",")
	want := "mw1-before,mw2-before,handler,mw2-after,mw1-after"
	if got != want {
		t.Fatalf("middleware order = %q, want %q", got, want)
	}
}

func TestGRPCHandlerFuncFallsBackToHTTP(t *testing.T) {
	served := false
	handler := grpcHandlerFunc(nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		served = true
		w.WriteHeader(http.StatusAccepted)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)

	if !served {
		t.Fatal("expected HTTP handler to serve request")
	}
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status code = %d, want %d", rec.Code, http.StatusAccepted)
	}
}

func TestInitGRPCDefaultsRuntimeMuxEndpointAndClientConn(t *testing.T) {
	svc, err := NewService(&Options{
		ServiceName: "svc",
		HttpPort:    18080,
		GrpcPort:    19090,
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	defer svc.closeClientConn()

	if err := svc.initGRPC(context.Background()); err != nil {
		t.Fatalf("initGRPC() error = %v", err)
	}

	if svc.options.RuntimeMuxEndpoint != defaultRuntimeMuxPath {
		t.Fatalf("RuntimeMuxEndpoint = %q, want %q", svc.options.RuntimeMuxEndpoint, defaultRuntimeMuxPath)
	}
	if svc.ClientConn() == nil {
		t.Fatal("ClientConn() returned nil after initGRPC")
	}
	if svc.GRPCServer() == nil {
		t.Fatal("GRPCServer() returned nil after initGRPC")
	}
}

func TestInitGRPCTLSRequiresFiles(t *testing.T) {
	svc, err := NewService(&Options{
		ServiceName: "svc",
		TLSEnabled:  true,
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	err = svc.initGRPC(context.Background())
	if err == nil || !strings.Contains(err.Error(), "certificate or key file is missing") {
		t.Fatalf("initGRPC() error = %v, want missing certificate/key error", err)
	}
}
