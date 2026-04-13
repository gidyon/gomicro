package grpcauth

import (
	"context"
	"time"
)

func ExampleNewAPI() {
	api := NewAPI([]byte("secret"), "users", "users-api")
	api.AddAdminGroups(DefaultAdminGroup())
	api.AddSuperAdminGroups(DefaultSuperAdminGroup())

	token, err := api.GenToken(context.Background(), &Payload{
		ID:        "user-1",
		ProjectID: "project-1",
		Group:     DefaultUserGroup(),
	}, time.Now().Add(time.Hour))
	if err != nil {
		panic(err)
	}

	md, err := api.GetMetadataFromJwt(token)
	if err != nil {
		panic(err)
	}

	_ = md
}
