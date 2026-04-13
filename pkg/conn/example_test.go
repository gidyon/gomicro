package conn

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func ExampleOpenSql() {
	_, err := OpenSql(&DbOptions{
		Name:    "users",
		Address: "127.0.0.1:3306",
		User:    "root",
		Schema:  "users",
		ConnPool: &DbPoolSettings{
			MaxIdleConns: 5,
			MaxOpenConns: 20,
		},
	})
	fmt.Println(err == nil)
	// Output:
	// true
}

func ExampleDialGrpcService() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := DialGrpcService(ctx, &GrpcDialOptions{
		Address: "localhost:9090",
		DialOptions: []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		},
	})
	fmt.Println(err != nil)
	// Output:
	// true
}
