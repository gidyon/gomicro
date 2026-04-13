package gomicro

import (
	"net/http"

	"github.com/rs/zerolog"
)

func ExampleNewService() {
	svc, err := NewService(&Options{
		ServiceName: "users",
		HttpPort:    8080,
		GrpcPort:    9090,
		Logger:      NewLogger("users", zerolog.InfoLevel),
	})
	if err != nil {
		panic(err)
	}

	svc.AddEndpointFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func ExampleService_AddHTTPMiddlewares() {
	svc, err := NewService(nil)
	if err != nil {
		panic(err)
	}

	svc.AddHTTPMiddlewares(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Service", "gomicro")
			next.ServeHTTP(w, r)
		})
	})
}
